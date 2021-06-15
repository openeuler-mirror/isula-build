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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const (
	// Uncompressed represents uncompressed
	Uncompressed = archive.Uncompressed
)

// ExportOptions is a struct for exporter
type ExportOptions struct {
	SystemContext      *types.SystemContext
	Ctx                context.Context
	ReportWriter       io.Writer
	DataDir            string
	ExportID           string
	ManifestType       string
	ImageListSelection cp.ImageListSelection
}

// Export exports an image to an output destination
func Export(imageID, outputDest string, opts ExportOptions, localStore *store.Store) error {
	eLog := logrus.WithField(util.LogKeySessionID, opts.Ctx.Value(util.LogFieldKey(util.LogKeySessionID)))
	if outputDest == "" {
		return nil
	}
	epter, err := parseExporter(opts, imageID, outputDest, localStore)
	if err != nil {
		return err
	}
	defer epter.Remove(opts.ExportID)

	registry, err := util.ParseServer(outputDest)
	if err != nil {
		return err
	}
	opts.SystemContext.DockerCertPath, err = securejoin.SecureJoin(constant.DefaultCertRoot, registry)
	if err != nil {
		return err
	}

	ref, digest, err := export(epter, opts)
	if err != nil {
		return errors.Wrapf(err, "export image from %s to %s failed", imageID, outputDest)
	}
	if ref != nil {
		eLog.Debugf("Export image with reference %s", ref.Name())
	}
	eLog.Infof("Successfully output image with digest %s", digest.String())

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

func export(e Exporter, exOpts ExportOptions) (reference.Canonical, digest.Digest, error) {
	var (
		ref            reference.Canonical
		manifestBytes  []byte
		manifestDigest digest.Digest
	)

	cpOpts := NewCopyOptions(exOpts)
	policyContext, err := NewPolicyContext(exOpts.SystemContext)
	if err != nil {
		return nil, "", err
	}
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
		return nil, "", errors.Errorf("get dest or src reference by export ID %v failed %v", exOpts.ExportID, err)
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
	if e.Name() == "isulad" {
		tarPathRegexp := regexp.MustCompile("(.*).tar")
		tarPath := tarPathRegexp.FindString(e.GetDestRef(exOpts.ExportID).StringWithinTransport())
		if err := exportToIsulad(exOpts.Ctx, tarPath); err != nil {
			return nil, "", errors.Wrapf(err, "export to isulad failed")
		}
	}

	return ref, manifestDigest, nil
}

// parseExporter parses an exporter instance and inits it with the src and dest reference.
func parseExporter(opts ExportOptions, src, destSpec string, localStore *store.Store) (Exporter, error) {
	const partsNum = 2
	// 1. parse exporter
	parts := strings.SplitN(destSpec, ":", partsNum)
	if len(parts) != partsNum {
		return nil, errors.Errorf(`invalid dest spec %q, expected colon-separated exporter:reference`, destSpec)
	}

	ept := GetAnExporter(parts[0])
	if ept == nil {
		return nil, errors.Errorf(`invalid image name: %q, unknown exporter "%s"`, src, parts[0])
	}

	// 2. Init exporter reference
	err := ept.Init(opts, src, destSpec, localStore)
	if err != nil {
		return nil, errors.Wrap(err, `fail to Init exporter"`)
	}
	return ept, nil
}

// NewCopyOptions will return copy options
func NewCopyOptions(opts ExportOptions) *cp.Options {
	cpOpts := &cp.Options{}
	cpOpts.SourceCtx = opts.SystemContext
	cpOpts.DestinationCtx = opts.SystemContext
	cpOpts.ReportWriter = opts.ReportWriter
	cpOpts.ForceManifestMIMEType = opts.ManifestType
	cpOpts.ImageListSelection = opts.ImageListSelection

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

// CheckArchiveFormat used to check if save or load image format is either docker-archive or oci-archive
func CheckArchiveFormat(format string) error {
	switch format {
	case constant.DockerArchiveTransport, constant.OCIArchiveTransport:
		return nil
	default:
		return errors.New("wrong image format provided")
	}
}

// FormatTransport for formatting transport with corresponding path
func FormatTransport(transport, path string) string {
	if transport == constant.DockerTransport {
		return fmt.Sprintf("%s://%s", transport, path)
	}
	return fmt.Sprintf("%s:%s", transport, path)
}

// GetManifestType for choosing corresponding manifest type according to format provided
func GetManifestType(format string) (string, error) {
	var manifestType string
	switch format {
	case constant.OCITransport:
		manifestType = imgspecv1.MediaTypeImageManifest
	case constant.DockerTransport:
		manifestType = manifest.DockerV2Schema2MediaType
	default:
		return "", errors.Errorf("unknown format %q. Choose one of the supported formats: %s,%s", format, constant.DockerTransport, constant.OCITransport)
	}
	return manifestType, nil
}

