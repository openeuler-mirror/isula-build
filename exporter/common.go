// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu, Jingxiao Lu
// Create: 2020-03-20
// Description: exporter related common functions

// Package exporter is used to export images
package exporter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/stringid"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const (
	// Uncompressed represents uncompressed
	Uncompressed = archive.Uncompressed
)

// ExportOptions is a struct for exporter
type ExportOptions struct {
	SystemContext *types.SystemContext
	Ctx           context.Context
	ReportWriter  io.Writer
	DataDir       string
	ExportID      string
	ManifestType  string
}

// Export export an archive to the client
func Export(src, destSpec string, opts ExportOptions, localStore *store.Store) error {
	eLog := logrus.WithField(util.LogKeySessionID, opts.Ctx.Value(util.LogFieldKey(util.LogKeySessionID)))
	if destSpec == "" {
		return nil
	}
	epter, isuladTarPath, err := parseExporter(opts, src, destSpec, localStore)
	if err != nil {
		return err
	}
	defer epter.Remove(opts.ExportID)

	registry, err := util.ParseServer(destSpec)
	if err != nil {
		return err
	}
	opts.SystemContext.DockerCertPath = filepath.Join(constant.DefaultCertRoot, registry)

	options := NewCopyOptions(opts)

	policyContext, err := NewPolicyContext(opts.SystemContext)
	if err != nil {
		return err
	}
	ref, digest, err := export(opts, epter, policyContext, options)
	if err != nil {
		return errors.Errorf("export image from %s to %s failed, got error: %s", src, destSpec, err)
	}
	if ref != nil {
		eLog.Debugf("Export image with reference %s", ref.Name())
	}
	eLog.Infof("Successfully output image with digest %s", digest.String())

	if err := exportToIsulad(opts.Ctx, isuladTarPath); err != nil {
		eLog.Error(err)
		return err
	}

	return nil
}

func exportToIsulad(ctx context.Context, tarPath string) error {
	// no tarPath need to export
	if len(tarPath) == 0 {
		return nil
	}
	defer func() {
		if rErr := os.Remove(tarPath); rErr != nil {
			logrus.Errorf("Remove file %s failed: %v", tarPath, rErr)
		}
	}()
	// dest here will not be influenced by external input, no security risk
	cmd := exec.CommandContext(ctx, "isula", "load", "-i", tarPath) // nolint:gosec
	if bytes, lErr := cmd.CombinedOutput(); lErr != nil {
		logrus.Errorf("Load image to isulad failed, stderr: %v, err: %v", string(bytes), lErr)
		return errors.Errorf("load image to isulad failed, stderr: %v, err: %v", string(bytes), lErr)
	}

	return nil
}

func export(exOpts ExportOptions, e Exporter, policyContext *signature.PolicyContext, cpOpts *cp.Options) (reference.Canonical, digest.Digest, error) {
	var (
		err            error
		ref            reference.Canonical
		manifestBytes  []byte
		manifestDigest digest.Digest
	)
	defer func() {
		destroyErr := policyContext.Destroy()
		if err == nil {
			err = destroyErr
		} else {
			err = errors.Wrapf(err, "destroy policy context got error: %v", destroyErr)
		}
	}()

	destRef, srcRef := e.GetDestRef(exOpts.ExportID), e.GetSrcRef(exOpts.ExportID)
	if destRef == nil || srcRef == nil {
		return nil, "", errors.Wrapf(err, "get dest or src reference by export ID %v failed", exOpts.ExportID)
	}
	if manifestBytes, err = cp.Image(exOpts.Ctx, policyContext, destRef, srcRef, cpOpts); err != nil {
		return nil, "", errors.Wrap(err, "copying layers and metadata failed")
	}
	if manifestDigest, err = manifest.Digest(manifestBytes); err != nil {
		return nil, "", errors.Wrap(err, "computing digest of manifest of new image failed")
	}
	if name := destRef.DockerReference(); name != nil {
		ref, err = reference.WithDigest(name, manifestDigest)
		if err != nil {
			return nil, "", errors.Wrapf(err, "generating canonical reference with name %q and digest %s failed", name, manifestDigest.String())
		}
	}
	return ref, manifestDigest, nil
}

// parseExporter parses an exporter instance and inits it with the src and dest reference.
func parseExporter(opts ExportOptions, src, destSpec string, localStore *store.Store) (Exporter, string, error) {
	const partsNum = 2
	var isuladTarPath string
	// 1. parse exporter
	parts := strings.SplitN(destSpec, ":", partsNum)
	if len(parts) != partsNum {
		return nil, "", errors.Errorf(`invalid dest spec %q, expected colon-separated exporter:reference`, destSpec)
	}

	ept := GetAnExporter(parts[0])
	if ept == nil {
		return nil, "", errors.Errorf(`invalid image name: %q, unknown exporter "%s"`, src, parts[0])
	}

	// 2. get src reference
	srcReference, _, err := image.FindImage(localStore, src)
	if err != nil {
		return nil, "", errors.Errorf("find src image: %q failed, got error: %v", src, err)
	}

	// 3. get dest reference
	if parts[0] == "isulad" {
		randomID := stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]
		isuladTarPath = filepath.Join(opts.DataDir, fmt.Sprintf("isula-build-tmp-%s.tar", randomID))
		// construct format: transport:path:image:tag
		// parts[1] here could not be empty cause client-end already processed it
		destSpec = fmt.Sprintf("docker-archive:%s:%s", isuladTarPath, parts[1])
		logrus.Infof("Process isulad output %s", destSpec)
	}
	destReference, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return nil, "", errors.Errorf("parse dest spec: %q failed, got error: %v", destSpec, err)
	}

	// 4. init exporter with src reference and dest reference
	ept.Init(opts.ExportID, srcReference, destReference)

	return ept, isuladTarPath, nil
}

// NewCopyOptions will return copy options
func NewCopyOptions(opts ExportOptions) *cp.Options {
	cpOpts := &cp.Options{}
	cpOpts.SourceCtx = opts.SystemContext
	cpOpts.DestinationCtx = opts.SystemContext
	cpOpts.ReportWriter = opts.ReportWriter

	return cpOpts
}

// NewPolicyContext return policy context from system context
func NewPolicyContext(sc *types.SystemContext) (*signature.PolicyContext, error) {
	pushPolicy, err := signature.DefaultPolicy(sc)
	if err != nil {
		return nil, errors.Wrap(err, "error reading the policy file")
	}
	policyContext, err := signature.NewPolicyContext(pushPolicy)
	if err != nil {
		return nil, errors.Errorf("error creating new signature policy context")
	}

	return policyContext, nil
}

// PipeWrapper is a wrapper for a pipefile
type PipeWrapper struct {
	PipeFile string
	Done     bool
}

// Close set the done flag for this pip
func (p *PipeWrapper) Close() {
	p.Done = true
}

// NewPipeWrapper checks the exporter type and creates the pipeFile for local archive exporting if needed
func NewPipeWrapper(runDir, expt string) (*PipeWrapper, error) {
	pipeFile := filepath.Join(runDir, fmt.Sprintf("exporter-%s", expt))
	if err := syscall.Mkfifo(pipeFile, constant.DefaultRootFileMode); err != nil {
		return nil, err
	}
	pipeWraper := PipeWrapper{
		PipeFile: pipeFile,
	}
	return &pipeWraper, nil
}

// PipeArchiveStream pipes the GRPC stream with pipeFile
func PipeArchiveStream(pipeWrapper *PipeWrapper) (f *os.File, err error) {
	var file *os.File
	if pipeWrapper == nil || pipeWrapper.PipeFile == "" {
		return nil, errors.New("no pipe wrapper found")
	}

	if file, err = os.OpenFile(pipeWrapper.PipeFile, os.O_RDONLY, os.ModeNamedPipe); err != nil {
		return nil, err
	}
	return file, nil
}

// ArchiveRecv receive data stream and write to file
func ArchiveRecv(ctx context.Context, dest string, isIsulad bool, fc chan []byte) error {
	var (
		err error
	)
	fo, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := fo.Close(); cerr != nil {
			logrus.Errorf("Close file %s failed: %v", dest, cerr)
		}
		if err != nil || isIsulad {
			if rerr := os.Remove(dest); rerr != nil {
				logrus.Errorf("Remove file %s failed: %v", dest, rerr)
			}
		}
	}()

	if err = fo.Chmod(constant.DefaultRootFileMode); err != nil {
		return err
	}

	w := bufio.NewWriter(fo)
	for bytes := range fc {
		if _, werr := w.Write(bytes); werr != nil {
			return err
		}
	}

	if err = w.Flush(); err != nil {
		return errors.Errorf("flush buffer failed: %v", err)
	}

	if isIsulad {
		// dest here will not be influenced by external input, no security risk
		cmd := exec.CommandContext(ctx, "isula", "load", "-i", dest) // nolint:gosec
		if bytes, lerr := cmd.CombinedOutput(); lerr != nil {
			return errors.Errorf("load image to isulad failed, stderr: %v, err: %v", string(bytes), lerr)
		}
	}

	return nil
}
