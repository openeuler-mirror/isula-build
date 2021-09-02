// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
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

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	savedocker "isula.org/isula-build/exporter/docker/archive"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type savedImage struct {
	exist bool
	tags  []reference.NamedTagged
}

type saveOptions struct {
	sysCtx            *types.SystemContext
	localStore        *store.Store
	saveID            string
	format            string
	oriImgList        []string
	finalImageOrdered []string
	finalImageSet     map[string]*savedImage
	outputPath        string
	logger            *logger.Logger
	logEntry          *logrus.Entry
}

func (b *Backend) getSaveOptions(req *pb.SaveRequest) saveOptions {
	return saveOptions{
		sysCtx:            image.GetSystemContext(),
		localStore:        b.daemon.localStore,
		saveID:            req.GetSaveID(),
		format:            req.GetFormat(),
		oriImgList:        req.GetImages(),
		finalImageOrdered: make([]string, 0),
		finalImageSet:     make(map[string]*savedImage),
		outputPath:        req.GetPath(),
		logger:            logger.NewCliLogger(constant.CliLogBufferLen),
		logEntry:          logrus.WithFields(logrus.Fields{"SaveID": req.GetSaveID(), "Format": req.GetFormat()}),
	}
}

// Save receives a save request and save the image(s) into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) error {
	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
		"Format": req.GetFormat(),
	}).Info("SaveRequest received")

	var err error
	opts := b.getSaveOptions(req)

	if err = checkFormatAndExpandTag(&opts); err != nil {
		return err
	}
	if err = filterImageName(&opts); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rErr := os.Remove(opts.outputPath); rErr != nil && !os.IsNotExist(rErr) {
				opts.logEntry.Warnf("Removing save output tarball %q failed: %v", opts.outputPath, rErr)
			}
		}
	}()

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), opts.saveID)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(exportHandler(ctx, &opts))
	eg.Go(messageHandler(stream, opts.logger))

	if err = eg.Wait(); err != nil {
		opts.logEntry.Warnf("Save stream closed with: %v", err)
		return err
	}

	return nil
}

func exportHandler(ctx context.Context, opts *saveOptions) func() error {
	return func() error {
		defer func() {
			opts.logger.CloseContent()
			if savedocker.DockerArchiveExporter.GetArchiveWriter(opts.saveID) != nil {
				if cErr := savedocker.DockerArchiveExporter.GetArchiveWriter(opts.saveID).Close(); cErr != nil {
					opts.logEntry.Errorf("Close archive writer failed: %v", cErr)
				}
				savedocker.DockerArchiveExporter.RemoveArchiveWriter(opts.saveID)
			}
		}()

		for _, imageID := range opts.finalImageOrdered {
			copyCtx := *opts.sysCtx
			if opts.format == constant.DockerArchiveTransport {
				// It's ok for DockerArchiveAdditionalTags == nil, as a result, no additional tags will be appended to the final archive file.
				copyCtx.DockerArchiveAdditionalTags = opts.finalImageSet[imageID].tags
			}

			exOpts := exporter.ExportOptions{
				Ctx:           ctx,
				SystemContext: &copyCtx,
				ExportID:      opts.saveID,
				ReportWriter:  opts.logger,
			}

			if err := exporter.Export(imageID, exporter.FormatTransport(opts.format, opts.outputPath),
				exOpts, opts.localStore); err != nil {
				opts.logEntry.Errorf("Save image %q in format %q failed: %v", imageID, opts.format, err)
				return errors.Wrapf(err, "save image %q in format %q failed", imageID, opts.format)
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

func checkFormatAndExpandTag(opts *saveOptions) error {
	switch opts.format {
	case constant.DockerTransport:
		opts.format = constant.DockerArchiveTransport
	case constant.OCITransport:
		opts.format = constant.OCIArchiveTransport
	default:
		return errors.New("wrong image format provided")
	}

	for i, imageName := range opts.oriImgList {
		nameWithTag, err := image.CheckAndAddDefaultTag(imageName, opts.localStore)
		if err != nil {
			return errors.Wrapf(err, "check format and expand tag failed with image name %q", imageName)
		}
		opts.oriImgList[i] = nameWithTag
	}

	return nil
}

func filterImageName(opts *saveOptions) error {
	if opts.format == constant.OCIArchiveTransport {
		opts.finalImageOrdered = opts.oriImgList
		return nil
	}

	visitedImage := make(map[string]bool)
	for _, imageName := range opts.oriImgList {
		if _, exists := visitedImage[imageName]; exists {
			continue
		}
		visitedImage[imageName] = true

		_, img, err := image.FindImage(opts.localStore, imageName)
		if err != nil {
			return errors.Wrapf(err, "filter image name failed when finding image name %q", imageName)
		}

		finalImage, ok := opts.finalImageSet[img.ID]
		if !ok {
			finalImage = &savedImage{exist: true}
			finalImage.tags = []reference.NamedTagged{}
			opts.finalImageOrdered = append(opts.finalImageOrdered, img.ID)
		}

		ref, err := reference.Parse(imageName)
		if err != nil {
			return errors.Wrapf(err, "filter image name failed when parsing name %q", imageName)
		}
		tagged, withTag := ref.(reference.NamedTagged)
		if withTag {
			finalImage.tags = append(finalImage.tags, tagged)
		}
		opts.finalImageSet[img.ID] = finalImage
	}

	return nil
}
