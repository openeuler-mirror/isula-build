// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-01-20
// Description: This file is used for "build" command

package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/storage/pkg/stringid"
	"github.com/gogo/protobuf/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	_ "isula.org/isula-build/exporter/register"
	"isula.org/isula-build/pkg/opts"
	"isula.org/isula-build/util"
)

type buildOptions struct {
	file          string
	format        string
	output        string
	buildArgs     []string
	capAddList    []string
	contextDir    string
	buildID       string
	proxyFlag     bool
	buildStatic   opts.ListOpts
	imageIDFile   string
	additionalTag string
}

const (
	buildExample = `isula-build ctr-img build -f Dockerfile .
isula-build ctr-img build -f Dockerfile -o docker-archive:name.tar:image:tag .
isula-build ctr-img build -f Dockerfile -o docker-daemon:image:tag .
isula-build ctr-img build -f Dockerfile -o docker://registry.example.com/repository:tag .
isula-build ctr-img build -f Dockerfile -o isulad:image:tag .
isula-build ctr-img build -f Dockerfile --build-static='build-time=2020-06-30 15:05:05' .`
	// buildTimeType is an option for static-build
	buildTimeType = "build-time"
)

var buildOpts buildOptions = buildOptions{
	buildStatic: opts.NewListOpts(opts.OptValidator),
}

// NewContainerImageBuildCmd returns container image operations commands
func NewContainerImageBuildCmd() *cobra.Command {
	ctrImgBuildCmd := &cobra.Command{
		Use:   "ctr-img",
		Short: "Container Image Operations",
	}
	ctrImgBuildCmd.AddCommand(
		NewBuildCmd(),
		NewImagesCmd(),
		NewRemoveCmd(),
		NewLoadCmd(),
		NewPullCmd(),
		NewPushCmd(),
		NewImportCmd(),
		NewTagCmd(),
		NewSaveCmd(),
	)

	disableFlags(ctrImgBuildCmd)

	return ctrImgBuildCmd
}

// NewBuildCmd cmd for container image building
func NewBuildCmd() *cobra.Command {
	// buildCmd represents the "build" command
	buildCmd := &cobra.Command{
		Use:     "build [FLAGS] PATH",
		Short:   "Build container images",
		Example: buildExample,
		RunE:    buildCommand,
	}

	buildCmd.PersistentFlags().StringVarP(&buildOpts.file, "filename", "f", "", "Path for Dockerfile")
	if util.CheckCliExperimentalEnabled() {
		buildCmd.PersistentFlags().StringVar(&buildOpts.format, "format", "oci", "Image format of the built image")
	} else {
		buildOpts.format = exporter.DockerTransport
	}
	buildCmd.PersistentFlags().StringVarP(&buildOpts.output, "output", "o", "", "Destination of output images")
	buildCmd.PersistentFlags().BoolVar(&buildOpts.proxyFlag, "proxy", true, "Inherit proxy environment variables from host")
	buildCmd.PersistentFlags().Var(&buildOpts.buildStatic, "build-static", "Static build with the given option")
	buildCmd.PersistentFlags().StringArrayVar(&buildOpts.buildArgs, "build-arg", []string{}, "Arguments used during build time")
	buildCmd.PersistentFlags().StringArrayVar(&buildOpts.capAddList, "cap-add", []string{}, "Add Linux capabilities for RUN command")
	buildCmd.PersistentFlags().StringVar(&buildOpts.imageIDFile, "iidfile", "", "Write image ID to the file")
	buildCmd.PersistentFlags().StringVarP(&buildOpts.additionalTag, "tag", "t", "", "Add tag to the built image")

	return buildCmd
}

func buildCommand(c *cobra.Command, args []string) error {
	if err := newBuildOptions(args); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		imageID, err2 := runBuild(ctx, cli)
		if err2 != nil {
			logrus.Debugf("Build failed: %v", err2)
			cancel()
		} else {
			logrus.Debugf("Build success with image id: %s", imageID)
		}
		return errors.Wrap(err2, "error runBuild")
	})

	eg.Go(func() error {
		err2 := runStatus(ctx, cli)
		if err2 != nil {
			logrus.Debugf("Status get failed: %v", err2)
			cancel()
		}
		// user should not pay attention on runStatus error
		// the errors were already printed to daemon log if any
		return nil
	})

	return eg.Wait()
}

func newBuildOptions(args []string) error {
	// unique buildID for each build progress
	buildOpts.buildID = stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]

	if len(args) < 1 {
		// use current working directory as default context directory
		contextDir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "unable to choose current working directory as build context")
		}
		realPath, err := filepath.EvalSymlinks(contextDir)
		if err != nil {
			return errors.Wrapf(err, "error getting the real path from %q", contextDir)
		}
		buildOpts.contextDir = realPath
		return nil
	}

	// check cap list
	for _, c := range buildOpts.capAddList {
		if !util.CheckCap(c) {
			return errors.Errorf("cap %v is invalid", c)
		}
	}

	// the path may be a symbol link
	contextDir, err := filepath.Abs(args[0])
	if err != nil {
		return errors.Wrapf(err, "error deriving an absolute path from %q", args[0])
	}
	realPath, err := filepath.EvalSymlinks(contextDir)
	if err != nil {
		return errors.Wrapf(err, "error getting the real path from %q", contextDir)
	}
	f, err := os.Stat(realPath)
	if err != nil {
		return errors.Wrapf(err, "stat context directory path %q err", realPath)
	}
	if !f.IsDir() {
		return errors.Errorf("context directory path %q should be a directory", realPath)
	}
	buildOpts.contextDir = realPath
	return nil
}

func checkOutput(output string) ([]string, error) {
	const outputFieldLen = 2
	const longestOutputLen = 512
	// just build, no export action
	if len(output) == 0 {
		return nil, nil
	}
	if len(output) > longestOutputLen {
		return nil, errors.Errorf("output should not longer than %v", longestOutputLen)
	}
	segments := strings.Split(output, ":")
	transport := segments[0]
	if len(transport) == 0 {
		return nil, errors.New("transport should not be empty")
	}
	if !exporter.IsSupport(transport) {
		return nil, errors.Errorf("transport %q not support", transport)
	}

	if len(segments) < outputFieldLen || strings.TrimSpace(segments[1]) == "" {
		return nil, errors.New("destination should not be empty")
	}

	return segments, nil
}

func checkAbsPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		pwd, err := os.Getwd()
		if err != nil {
			return "", errors.New("get current path failed")
		}
		path = util.MakeAbsolute(path, pwd)
	}
	if util.IsExist(path) {
		return "", errors.Errorf("output file already exist: %q, try to remove existing tarball or rename output file", path)
	}

	return path, nil
}

func modifyLocalTransporter(transport string, absPath string, segments []string) error {
	const validIsuladFieldsLen = 3
	switch transport {
	case exporter.DockerArchiveTransport, exporter.OCIArchiveTransport:
		newSeg := util.CopyStrings(segments)
		newSeg[1] = absPath
		buildOpts.output = strings.Join(newSeg, ":")
		return nil
	case exporter.IsuladTransport:
		if len(segments) != validIsuladFieldsLen {
			return errors.Errorf("invalid isulad output format: %v", buildOpts.output)
		}
		return nil
	default:
		return nil
	}
}

func checkAndProcessOutput() error {
	// validate output
	segments, err := checkOutput(buildOpts.output)
	if err != nil {
		return err
	}
	if segments == nil {
		return nil
	}

	transport := segments[0]
	// just build, not need to export to any destination
	if !exporter.IsClientExporter(transport) {
		return nil
	}

	// segments here could not be nil, so just get the path from it
	outputAbsPath, cErr := checkAbsPath(segments[1])
	if cErr != nil {
		return cErr
	}

	// according to transport, we modify them by changing output path
	if mErr := modifyLocalTransporter(transport, outputAbsPath, segments); mErr != nil {
		return mErr
	}

	return nil
}

func parseStaticBuildOpts() (*pb.BuildStatic, time.Time, error) {
	var (
		err         error
		t           = time.Now()
		buildStatic = &pb.BuildStatic{}
	)
	for mode, v := range buildOpts.buildStatic.Values {
		switch mode {
		case buildTimeType:
			if t, err = time.Parse(constant.LayoutTime, v); err != nil {
				return nil, t, errors.Wrap(err, "build time format need like '2020-05-23 10:55:33'")
			}
			if buildStatic.BuildTime, err = types.TimestampProto(t); err != nil {
				return nil, t, err
			}
		default:
			return nil, t, errors.Errorf("option %q not support by build-static", mode)
		}
	}

	return buildStatic, t, nil
}

func runBuild(ctx context.Context, cli Cli) (string, error) {
	var (
		encrypted       bool
		err             error
		content         string
		imageIDFilePath string
		digest          string
	)

	if err = exporter.CheckImageFormat(buildOpts.format); err != nil {
		return "", err
	}
	if err = checkAndProcessOutput(); err != nil {
		return "", err
	}
	if content, digest, err = readDockerfile(); err != nil {
		return "", err
	}
	if encrypted, err = encryptBuildArgs(util.DefaultRSAKeyPath); err != nil {
		return "", errors.Wrap(err, "encrypt --build-arg failed")
	}
	if imageIDFilePath, err = getAbsPath(buildOpts.imageIDFile); err != nil {
		return "", err
	}
	buildOpts.imageIDFile = imageIDFilePath

	buildStatic, t, err := parseStaticBuildOpts()
	if err != nil {
		return "", err
	}
	entityID := fmt.Sprintf("%s:%s", digest, t.String())

	buildResp, err := cli.Client().Build(ctx, &pb.BuildRequest{
		BuildType:     constant.BuildContainerImageType,
		BuildID:       buildOpts.buildID,
		EntityID:      entityID,
		BuildArgs:     buildOpts.buildArgs,
		CapAddList:    buildOpts.capAddList,
		ContextDir:    buildOpts.contextDir,
		FileContent:   content,
		Output:        buildOpts.output,
		Proxy:         buildOpts.proxyFlag,
		BuildStatic:   buildStatic,
		Iidfile:       buildOpts.imageIDFile,
		AdditionalTag: buildOpts.additionalTag,
		Encrypted:     encrypted,
		Format:        buildOpts.format,
	})
	if err != nil {
		return "", err
	}

	return buildResp.ImageID, err
}

// encrypts those sensitive args before transporting via GRPC
func encryptBuildArgs(path string) (bool, error) {
	var hasSensiArg bool
	for _, v := range buildOpts.buildArgs {
		const kvNums = 2
		var ss = strings.SplitN(v, "=", kvNums)
		// check whether there is sensitive build-arg, if has, goto encrypt all build-args
		if constant.ReservedArgs[ss[0]] {
			hasSensiArg = true
			break
		}
	}
	if !hasSensiArg {
		return hasSensiArg, nil
	}

	key, err := util.ReadPublicKey(path)
	if err != nil {
		return hasSensiArg, err
	}

	const possibleArgCaps = 10
	var args = make([]string, 0, possibleArgCaps)
	for _, v := range buildOpts.buildArgs {
		encryptedArg, encErr := util.EncryptRSA(v, key, sha512.New())
		if encErr != nil {
			return hasSensiArg, encErr
		}
		args = append(args, encryptedArg)
	}

	buildOpts.buildArgs = args
	return hasSensiArg, nil
}

func runStatus(ctx context.Context, cli Cli) error {
	status, err := cli.Client().Status(ctx, &pb.StatusRequest{
		BuildID: buildOpts.buildID,
	})
	if err != nil {
		return err
	}

	for {
		msg, err := status.Recv()
		if msg != nil {
			fmt.Print(msg.Content)
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// readDockerfile validates the --file, opens it and returns its content and sha256sum
// The possible Dockerfile path should be: filepath or contextDir+filepath
// or contextDir+Dockerfile if filepath is empty
func readDockerfile() (string, string, error) {
	resolvedPath, err := resolveDockerfilePath()
	if err != nil {
		return "", "", err
	}

	f, err := os.Open(filepath.Clean(resolvedPath))
	if err != nil {
		return "", "", errors.Wrapf(err, "open dockerfile failed")
	}
	defer func() {
		if err2 := f.Close(); err2 != nil {
			logrus.Warnf("Close dockerfile %s failed", resolvedPath)
		}
	}()

	srcHasher := digest.Canonical.Digester()
	reader := io.TeeReader(f, srcHasher.Hash())

	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", "", errors.Wrapf(err, "read dockerfile failed")
	}
	hash := srcHasher.Digest().String()
	parts := strings.SplitN(hash, ":", 2)
	logrus.Debugf("Read Dockerfile at %s", resolvedPath)
	return string(buf), parts[1], nil
}

func resolveDockerfilePath() (string, error) {
	var resolvedPath = buildOpts.file

	if buildOpts.file == "" {
		// filepath is empty, try to resolve with contextDir+Dockerfile
		resolvedPath = path.Join(buildOpts.contextDir, "Dockerfile")
	}
	// stat path with origin filepath or contextDir+Dockerfile
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		logrus.Debugf("Stat dockerfile failed with path %s", resolvedPath)
		// not found with filepath, try to resolve with contextDir+filepath
		resolvedPath = path.Join(buildOpts.contextDir, buildOpts.file)
		fileInfo, err = os.Stat(resolvedPath)
		if err != nil {
			logrus.Debugf("Stat dockerfile failed again with path %s", resolvedPath)
			return "", errors.Wrapf(err, "stat dockerfile failed with filename %s", buildOpts.file)
		}
	}
	if !fileInfo.Mode().IsRegular() {
		return "", errors.Errorf("file %s should be a regular file", resolvedPath)
	}
	if fileInfo.Size() == 0 {
		return "", errors.New("file is empty, is it a normal dockerfile?")
	}
	if fileInfo.Size() > constant.MaxFileSize {
		return "", errors.Errorf("file is too big with size %v, is it a normal dockerfile?", fileInfo.Size())
	}
	return resolvedPath, nil
}

func getAbsPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return util.MakeAbsolute(path, pwd), nil
}
