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
// Create: 2020-03-20
// Description: Builder related functions

package dockerfile

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/containers/image/v5/docker/reference"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	dockerfile "isula.org/isula-build/builder/dockerfile/parser"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

// BuildOptions is the option for build an image
type BuildOptions struct {
	BuildArgs     map[string]string
	ContextDir    string
	File          string
	Iidfile       string
	Output        []string
	CapAddList    []string
	ProxyFlag     bool
	Tag           string
	AdditionalTag string
}

// Builder is the object to build a Dockerfile
type Builder struct {
	cliLog           *logger.Logger
	playbook         *parser.PlayBook
	buildID          string
	entityID         string
	ctx              context.Context
	localStore       *store.Store
	buildOpts        BuildOptions
	runtimePath      string
	dataDir          string
	runDir           string
	dockerfileDigest string
	buildTime        *time.Time
	ignores          []string
	headingArgs      map[string]string
	reservedArgs     map[string]string
	unusedArgs       map[string]string
	stageBuilders    []*stageBuilder
	// stageAliasMap hold the stage index which has been renamed
	// e.g. FROM foo AS bar  ->  map[string]int{"bar":1}
	stageAliasMap map[string]int
	rsaKey        *rsa.PrivateKey
}

// NewBuilder init a builder
func NewBuilder(ctx context.Context, store *store.Store, req *pb.BuildRequest, runtimePath, buildDir, runDir string, key *rsa.PrivateKey) (*Builder, error) {
	b := &Builder{
		ctx:          ctx,
		buildID:      req.BuildID,
		entityID:     req.EntityID,
		cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
		unusedArgs:   make(map[string]string),
		headingArgs:  make(map[string]string),
		reservedArgs: make(map[string]string),
		localStore:   store,
		runtimePath:  runtimePath,
		dataDir:      buildDir,
		runDir:       runDir,
		rsaKey:       key,
	}

	args, err := b.parseBuildArgs(req.GetBuildArgs(), req.GetEncrypted())
	if err != nil {
		return nil, errors.Wrap(err, "parse build-arg failed")
	}

	for _, c := range req.GetCapAddList() {
		if !util.CheckCap(c) {
			return nil, errors.Errorf("cap %v is invalid", c)
		}
	}

	b.buildOpts = BuildOptions{
		ContextDir: req.GetContextDir(),
		File:       req.GetFileContent(),
		BuildArgs:  args,
		CapAddList: req.GetCapAddList(),
		ProxyFlag:  req.GetProxy(),
		Iidfile:    req.GetIidfile(),
		Output:     []string{req.GetOutput()},
	}
	b.parseStaticBuildOpts(req)
	tag, additionalTag, err := parseTag(req.Output, req.AdditionalTag)
	if err != nil {
		return nil, err
	}
	b.buildOpts.Tag, b.buildOpts.AdditionalTag = tag, additionalTag

	// prepare workdirs for dockerfile builder
	for _, dir := range []string{buildDir, runDir} {
		if err = os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
			return nil, err
		}

		defer func(dir string) {
			if err != nil {
				if rerr := os.RemoveAll(dir); rerr != nil {
					logrus.WithField(util.LogKeySessionID, b.buildID).
						Warnf("Removing dir in rollback failed: %v", rerr)
				}
			}
		}(dir)
	}

	return b, nil
}

func parseTag(output, additionalTag string) (string, string, error) {
	var (
		err    error
		tag    string
		addTag string
	)
	if tag = parseOutputTag(output); tag != "" {
		_, tag, err = CheckAndExpandTag(tag)
		if err != nil {
			return "", "", err
		}
	}

	if additionalTag != "" {
		_, addTag, err = CheckAndExpandTag(additionalTag)
		if err != nil {
			return "", "", err
		}
	}

	return tag, addTag, nil
}

// Logger adds the "buildID" attribute to build logs
func (b *Builder) Logger() *logrus.Entry {
	return logrus.WithField(util.LogKeySessionID, b.ctx.Value(util.LogFieldKey(util.LogKeySessionID)))
}

func (b *Builder) parseBuildArgs(buildArgs []string, encrypted bool) (map[string]string, error) {
	args := make(map[string]string, len(buildArgs))
	for _, arg := range buildArgs {
		if encrypted {
			v, err := util.DecryptRSA(arg, b.rsaKey, crypto.SHA512)
			if err != nil {
				return nil, err
			}
			arg = v
		}
		kv := strings.SplitN(arg, "=", 2)
		if len(kv) > 1 {
			args[kv[0]] = kv[1]
		}
	}
	return args, nil
}

func (b *Builder) parseStaticBuildOpts(req *pb.BuildRequest) {
	if buildStatic := req.GetBuildStatic(); buildStatic != nil {
		t := buildStatic.GetBuildTime()
		if buildTime, err := time.Parse(time.RFC3339, t.String()); err == nil {
			b.buildTime = &buildTime
		}
	}
}

func (b *Builder) parseFiles() error {
	p, err := parser.NewParser(parser.DefaultParser)
	if err != nil {
		return errors.Wrap(err, "create parser failed")
	}

	srcHasher := digest.Canonical.Digester()
	rc := bytes.NewBufferString(b.buildOpts.File)
	reader := io.TeeReader(rc, srcHasher.Hash())
	playbook, err := p.Parse(reader, false)
	if err != nil {
		return errors.Wrap(err, "parse dockerfile failed")
	}
	hash := srcHasher.Digest().String()
	parts := strings.SplitN(hash, ":", 2)
	b.dockerfileDigest = parts[1]
	if playbook.Warnings != nil {
		warn := fmt.Sprintf("Parse dockerfile got warnings: %v\n", playbook.Warnings)
		b.Logger().Warnf(warn)
		b.cliLog.Print(warn)
	}
	b.playbook = playbook

	ignores, err := p.ParseIgnore(b.buildOpts.ContextDir)
	if err != nil {
		return errors.Wrap(err, "parse .dockerignore failed")
	}
	b.ignores = ignores

	return nil
}

func (b *Builder) newStageBuilders() error {
	var err error
	// 1. analyze the ARGs before first FROM command
	if err = b.usedHeadingArgs(); err != nil {
		return errors.Wrapf(err, "resolve heading ARGs failed")
	}

	// 2. loop stages for analyzing FROM command and creating StageBuilders
	b.stageAliasMap = make(map[string]int, len(b.playbook.Pages))
	for stageIdx, stage := range b.playbook.Pages {
		// new stage and analyze from command
		sb := newStageBuilder(stageIdx, stage.Name)
		if sb.fromImage, sb.fromStageIdx, err = analyzeFrom(stage.Lines[0], stageIdx, b.stageAliasMap, b.searchArg); err != nil {
			return err
		}
		sb.rawStage = stage
		sb.builder = b
		sb.env = make(map[string]string)
		sb.localStore = b.localStore

		// get registry from "fromImage"
		server, err := util.ParseServer(sb.fromImage)
		if err != nil {
			return err
		}
		sb.buildOpt.systemContext.DockerCertPath, err = securejoin.SecureJoin(constant.DefaultCertRoot, server)
		if err != nil {
			return err
		}

		b.stageBuilders = append(b.stageBuilders, sb)
	}

	return nil
}

// usedHeadingArgs check heading args with inputted build-args
// if the HeadingArg without default value doesn't matched build-args, not effects in this building;
// if the HeadingArg with default value doesn't matched build-args, effects with default value;
// if the HeadingArg with default value matched build-args, effects with the value specified by build-args
func (b *Builder) usedHeadingArgs() error {
	var (
		buildArgs   = util.CopyMapStringString(b.buildOpts.BuildArgs)
		headingArgs = make(map[string]string, len(b.playbook.HeadingArgs))
		reserved    = make(map[string]string, len(constant.ReservedArgs))
		resolveArg  = func(s string) string {
			if v, ok := headingArgs[s]; ok {
				return v
			}
			return ""
		}
	)

	for _, s := range b.playbook.HeadingArgs {
		kv := strings.Split(s, "=")
		// try word expansion for k in headingArgs. after resolved, replace it with new
		k, err := dockerfile.ResolveParam(kv[0], false, resolveArg)
		if err != nil {
			return errors.Wrapf(err, "word expansion for heading ARG %q failed", kv[0])
		}

		buildArg, inBuildArgs := buildArgs[k]
		if inBuildArgs {
			// if this heading arg is activated by --build-arg, assign it to headingArgs
			// and this buildArgs is used in this building, delete it from buildArgs (those not deleted are unusedArgs)
			headingArgs[k] = buildArg
			delete(buildArgs, k)
		} else {
			if len(kv) < 2 {
				// this heading ARG doesn't have default value and not activated by build-arg, not use for this build
				continue
			}
			// try word expansion for v in headingArgs
			v, err := dockerfile.ResolveParam(kv[1], false, resolveArg)
			if err != nil {
				return errors.Wrapf(err, "word expansion for heading ARG %q failed", s)
			}
			headingArgs[k] = v
		}
	}
	for k, v := range buildArgs {
		if constant.ReservedArgs[k] {
			reserved[k] = v
			delete(buildArgs, k)
		}
	}
	b.unusedArgs = util.CopyMapStringString(buildArgs)
	b.reservedArgs = reserved
	b.headingArgs = headingArgs

	return nil
}

func (b *Builder) searchArg(arg string) string {
	// supports the standard bash modifies as ${variable:-word} and ${variable:+word}
	if strings.Contains(arg, ":-") {
		subs := strings.Split(arg, ":-")
		if v, exist := b.headingArgs[subs[0]]; exist {
			delete(b.unusedArgs, arg)
			return v
		}
		if len(subs) < 2 {
			return ""
		}
		return subs[1]
	}
	if strings.Contains(arg, ":+") {
		subs := strings.Split(arg, ":+")
		if _, exist := b.headingArgs[subs[0]]; !exist || len(subs) < 2 {
			return ""
		}
		return subs[1]
	}

	// only accepts heading args when parsing params in FROM command
	if v, exist := b.headingArgs[arg]; exist {
		delete(b.unusedArgs, arg)
		return v
	}
	return ""
}

func analyzeFrom(line *parser.Line, stageIdx int, stageMap map[string]int, resolveArg func(string) string) (string, int, error) {
	fromImage, err := image.ResolveImageName(line.Cells[0].Value, resolveArg)
	if err != nil {
		return "", 0, err
	}

	fromStageIdx := -1
	if idx, exist := stageMap[fromImage]; exist {
		fromStageIdx = idx
	}
	// if this command is form "FROM foo AS bar" (3 is length without command name FROM)
	// which means this stage will be used later, mark it
	if len(line.Cells) == 3 {
		stageName := line.Cells[2].Value
		stageMap[stageName] = stageIdx
	}
	return fromImage, fromStageIdx, nil
}

func getFlagsAndArgs(line *parser.Line, allowFlags map[string]bool) (map[string]string, []string) {
	args := make([]string, 0, len(line.Cells))
	for _, c := range line.Cells {
		args = append(args, c.Value)
	}

	flags := make(map[string]string, len(line.Flags))
	for flag, value := range line.Flags {
		if _, ok := allowFlags[flag]; ok {
			flags[flag] = value
		}
	}

	return flags, args
}

// Build makes the image
func (b *Builder) Build() (string, error) {
	var (
		executeTimer = b.cliLog.StartTimer("\nTotal")
		err          error
		imageID      string
	)

	// 6. defer cleanup
	defer func() {
		b.cleanup()
	}()

	// 1. parseFiles
	if err = b.parseFiles(); err != nil {
		return "", err
	}

	// 2. pre-handle Playbook
	if err = b.newStageBuilders(); err != nil {
		return "", err
	}

	// 3. loop StageBuilders for building
	for _, stage := range b.stageBuilders {
		stageTimer := b.cliLog.StartTimer(fmt.Sprintf("Stage %d", stage.position))
		// update FROM from name to imageID if it is based on previous stage
		if idx := stage.fromStageIdx; idx != -1 {
			stage.fromImage = b.stageBuilders[idx].imageID
		}

		imageID, err = stage.stageBuild(b.ctx)
		b.cliLog.StopTimer(stageTimer)
		b.Logger().Debugln(b.cliLog.GetCmdTime(stageTimer))
		if err != nil {
			b.Logger().Errorf("Builder[%s] build for stage[%s] failed for: %v", b.buildID, stage.name, err)
			return "", errors.Wrapf(err, "building image for stage[%s] failed", stage.name)
		}
	}

	// 4. export images
	if err = b.export(imageID); err != nil {
		return "", errors.Wrapf(err, "exporting images failed")
	}

	// 5. output imageID
	if err = b.writeImageID(imageID); err != nil {
		return "", errors.Wrapf(err, "writing image ID failed")
	}

	b.cliLog.StopTimer(executeTimer)
	b.Logger().Debugf("Time Cost:\n%s", b.cliLog.Summary())
	return imageID, nil
}

func (b *Builder) cleanup() {
	// 1. warn user about the unused build-args if has
	if len(b.unusedArgs) != 0 {
		var unused []string
		for k := range b.unusedArgs {
			unused = append(unused, k)
		}
		sort.Strings(unused)
		b.cliLog.Print("[Warning] One or more build-args %v were not consumed\n", unused)
	}

	// 2. cleanup the stage resources
	for _, stage := range b.stageBuilders {
		if err := stage.delete(); err != nil {
			b.Logger().Warnf("Failed to cleanup stage resources for stage %q: %v", stage.name, err)
		}
	}

	// 3. close channel for status
	b.cliLog.CloseContent()
}

func (b *Builder) export(imageID string) error {
	exportTimer := b.cliLog.StartTimer("EXPORT")
	if err := b.applyTag(imageID); err != nil {
		return err
	}

	var retErr error
	for _, o := range b.buildOpts.Output {
		exOpts := exporter.ExportOptions{
			Ctx:           b.ctx,
			SystemContext: image.GetSystemContext(),
			ReportWriter:  b.cliLog,
			ExportID:      b.buildID,
			DataDir:       b.dataDir,
		}
		if exErr := exporter.Export(imageID, o, exOpts, b.localStore); exErr != nil {
			b.Logger().Errorf("Image %s output to %s failed with: %v", imageID, o, exErr)
			retErr = exErr
			continue
		}
		b.Logger().Infof("Image %s output to %s completed", imageID, o)
	}
	b.cliLog.StopTimer(exportTimer)
	b.Logger().Debugln(b.cliLog.GetCmdTime(exportTimer))
	return retErr
}

func (b *Builder) applyTag(imageID string) error {
	tags := make([]string, 0, 0)
	if b.buildOpts.Tag != "" {
		tags = append(tags, b.buildOpts.Tag)
	}
	if b.buildOpts.AdditionalTag != "" {
		tags = append(tags, b.buildOpts.AdditionalTag)
	}

	if len(tags) > 0 {
		if serr := b.localStore.SetNames(imageID, tags); serr != nil {
			return errors.Wrapf(serr, "set tags %v for image %v error", tags, imageID)
		}
	}

	return nil
}

func (b *Builder) writeImageID(imageID string) error {
	if b.buildOpts.Iidfile != "" {
		if err := ioutil.WriteFile(b.buildOpts.Iidfile, []byte(imageID), constant.DefaultRootFileMode); err != nil {
			b.Logger().Errorf("Write image ID [%s] to file [%s] failed: %v", imageID, b.buildOpts.Iidfile, err)
			return errors.Wrapf(err, "write image ID to file %s failed", b.buildOpts.Iidfile)
		}
		b.cliLog.Print("Write image ID [%s] to file: %s\n", imageID, b.buildOpts.Iidfile)
	} else {
		b.cliLog.Print("Build success with image id: %s\n", imageID)
	}
	return nil
}

// StatusChan return chan which contains build info of the builder
func (b *Builder) StatusChan() <-chan string {
	return b.cliLog.GetContent()
}

// CleanResources removes data dir and run dir of builder, and returns the last removing error
func (b *Builder) CleanResources() error {
	var err error
	for _, dir := range []string{b.dataDir, b.runDir} {
		if rerr := os.RemoveAll(dir); rerr != nil {
			b.Logger().Errorf("Removing working dir %q failed: %v", dir, rerr)
			err = rerr
		}
	}
	return err
}

// EntityID returns the entityID of the Builder
func (b *Builder) EntityID() string {
	return b.entityID
}

func parseOutputTag(output string) string {
	outputFields := strings.Split(output, ":")
	const archiveOutputWithoutTagLen = 2

	var tag string
	switch {
	case (outputFields[0] == "docker-daemon" || outputFields[0] == "isulad") && len(outputFields) > 1:
		tag = strings.Join(outputFields[1:], ":")
	case outputFields[0] == "docker-archive" && len(outputFields) > archiveOutputWithoutTagLen:
		tag = strings.Join(outputFields[archiveOutputWithoutTagLen:], ":")
	case outputFields[0] == "docker" && len(outputFields) > 1:
		repoAndTag := strings.Join(outputFields[1:], ":")
		// repo format regexp, "//registry.example.com/" for example
		repo := regexp.MustCompile(`^\/\/[\w\.\-\:]+\/`).FindString(repoAndTag)
		if repo == "" {
			return ""
		}
		tag = repoAndTag[len(repo):]
	}

	return tag
}

// CheckAndExpandTag checks tag name. If it not include a tag, "latest" will be added.
func CheckAndExpandTag(tag string) (reference.Named, string, error) {
	if tag == "" {
		return nil, "<none>:<none>", nil
	}

	newTag := tag
	slashLastIndex := strings.LastIndex(newTag, "/")
	sepLastIndex := strings.LastIndex(newTag, ":")
	if sepLastIndex == -1 || (sepLastIndex < slashLastIndex) {
		// isula
		// localhost:5000/isula
		newTag += ":latest"
	}

	const longestTagFieldsLen = 3
	if len(strings.Split(newTag, ":")) > longestTagFieldsLen {
		// localhost:5000:5000/isula:latest
		return nil, "", errors.Errorf("invalid tag: %v", newTag)
	}

	oriRef, err := reference.ParseNormalizedNamed(newTag)
	if err != nil {
		return nil, "", errors.Wrapf(err, "parse tag err, invalid tag: %v", newTag)
	}

	tagWithoutRepo := newTag[slashLastIndex+1:]
	_, err = reference.ParseNormalizedNamed(tagWithoutRepo)
	if err != nil {
		// isula:latest:latest
		// localhost/isula:latest:latest
		// isula!@#:latest
		// isula :latest
		return nil, "", errors.Wrapf(err, "parse tag err, invalid tag: %v", newTag)
	}

	return oriRef, newTag, nil
}
