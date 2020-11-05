// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// iSula-Kits licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-07-31
// Description: This file is "save" command for backend

package daemon

import (
	"context"
	"os"
	"strings"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/docker/reference"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/exporter"
	im "isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type imageInfo struct {
	image   *storage.Image
	tags    []reference.NamedTagged
	oriName string
}

type saveOptions struct {
	writer     *archive.Writer
	sysCtx     *types.SystemContext
	logger     *logger.Logger
	localStore *store.Store
	logEntry   *logrus.Entry
	images     map[string]*imageInfo
	saveID     string
	outputPath string
	oriImgList []string
}

func (b *Backend) getSaveOptions(req *pb.SaveRequest) saveOptions {
	return saveOptions{
		writer:     nil,
		sysCtx:     im.GetSystemContext(),
		logger:     logger.NewCliLogger(constant.CliLogBufferLen),
		localStore: b.daemon.localStore,
		images:     make(map[string]*imageInfo),
		saveID:     req.GetSaveID(),
		outputPath: req.GetPath(),
		oriImgList: req.GetImages(),
		logEntry:   logrus.WithFields(logrus.Fields{"SaveID": req.GetSaveID()}),
	}
}

// Save receives a save request and save the image(s) into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) (err error) {
	var (
		ok         bool
		archWriter *archive.Writer
	)
	opts := b.getSaveOptions(req)
	opts.logEntry.Info("SaveRequest received")

	archWriter, err = archive.NewWriter(opts.sysCtx, opts.outputPath)
	if err != nil {
		opts.logEntry.Error(err)
		return errors.Errorf("create archive writer failed: %v", err)
	}
	defer func() {
		if err != nil {
			if rErr := os.Remove(opts.outputPath); rErr != nil && !os.IsNotExist(rErr) {
				opts.logEntry.Warnf("Removing save output tarball %q failed: %v", opts.outputPath, rErr)
			}
		}
		if cErr := archWriter.Close(); cErr != nil {
			opts.logEntry.Errorf("Close archive writer failed: %v", cErr)
		}
	}()
	opts.writer = archWriter
	opts.images, err = getImagesFromLocal(opts)
	if err != nil {
		opts.logEntry.Error(err)
		return err
	}

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), opts.saveID)
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(exportHandler(ctx, stream, opts))
	eg.Go(messageHandler(stream, opts.logger))
	errC := make(chan error, 1)

	go func() { errC <- eg.Wait() }()
	defer close(errC)

	select {
	case err, ok = <-errC:
		if !ok {
			opts.logEntry.Info("Channel errC closed")
			return nil
		}
		if err != nil {
			return err
		}
	case _, ok := <-stream.Context().Done():
		if !ok {
			opts.logEntry.Info("Channel stream done closed")
			return nil
		}
		err = egCtx.Err()
		if err != nil && err != context.Canceled {
			opts.logEntry.Infof("Stream closed with: %v", err)
		}
	}

	return nil
}

func getImagesFromLocal(opts saveOptions) (map[string]*imageInfo, error) {
	var images = make(map[string]*imageInfo)
	for _, image := range opts.oriImgList {
		_, localImg, err := im.FindImageLocally(opts.localStore, image)
		if err != nil {
			opts.logEntry.Errorf("Lookup local image %s failed: %v", image, err)
			return nil, err
		}
		id := localImg.ID
		imgInfo, exists := images[id]
		if !exists {
			imgInfo = &imageInfo{image: localImg, oriName: image}
			images[id] = imgInfo
		}

		if ref, tag, err := checkTag(image, id); err == nil && tag != "" {
			refTag, ok := ref.(reference.NamedTagged)
			if !ok {
				tErr := errors.Errorf("invalid tag %s (%s): dose not contain a tag", tag, ref.String())
				opts.logEntry.Error(tErr)
				return nil, tErr
			}
			imgInfo.tags = append(imgInfo.tags, refTag)
		}
	}

	return images, nil
}

func exportHandler(ctx context.Context, stream pb.Control_SaveServer, opts saveOptions) func() error {
	return func() error {
		defer func() {
			opts.logger.CloseContent()
		}()

		policyContext, err := exporter.NewPolicyContext(opts.sysCtx)
		if err != nil {
			opts.logEntry.Errorf("Getting policy failed: %v", err)
			return errors.Wrap(err, "error getting policy")
		}
		defer func() {
			if err := policyContext.Destroy(); err != nil {
				opts.logEntry.Debugf("Error destroying signature policy context: %v", err)
			}
		}()

		for id, img := range opts.images {
			exOpts := exporter.ExportOptions{
				Ctx:           ctx,
				SystemContext: opts.sysCtx,
				ExportID:      opts.saveID,
				ReportWriter:  opts.logger,
			}
			src, err := is.Transport.NewStoreReference(opts.localStore, nil, id)
			if err != nil {
				opts.logEntry.Errorf("Getting source image %s failed: %v", img.oriName, err)
				return errors.Wrapf(err, "error getting source image ref for %q", img.oriName)
			}
			dest, errW := opts.writer.NewReference(nil)
			if errW != nil {
				opts.logEntry.Error(errW)
				return errW
			}
			copyOptions := exporter.NewCopyOptions(exOpts)
			copyOptions.DestinationCtx.DockerArchiveAdditionalTags = img.tags
			copyOptions.SourceCtx.DockerArchiveAdditionalTags = img.tags
			_, err = cp.Image(stream.Context(), policyContext, dest, src, copyOptions)
			if err != nil {
				opts.logEntry.Errorf("Copying source image %s failed: %v", img.oriName, err)
				return errors.Wrapf(err, "copying source image %s failed", img.oriName)
			}
		}

		return nil
	}
}

func messageHandler(stream pb.Control_SaveServer, cliLogger *logger.Logger) func() error {
	return func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err := stream.Send(&pb.SaveResponse{
				Log: content,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}

func checkTag(oriImg, imageID string) (reference.Named, string, error) {
	// no tag found
	if strings.HasPrefix(imageID, oriImg) {
		return nil, "", nil
	}
	ref, tag, err := dockerfile.CheckAndExpandTag(oriImg)
	if err != nil {
		return nil, "", err
	}
	return ref, tag, nil
}
