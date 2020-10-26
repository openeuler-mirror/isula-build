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
	images     map[string]*imageInfo
	saveID     string
	outputPath string
	oriImgList []string
}

func (b *Backend) getSaveOption(req *pb.SaveRequest) saveOptions {
	return saveOptions{
		writer:     nil,
		sysCtx:     im.GetSystemContext(),
		logger:     logger.NewCliLogger(constant.CliLogBufferLen),
		localStore: b.daemon.localStore,
		images:     make(map[string]*imageInfo),
		saveID:     req.GetSaveID(),
		outputPath: req.GetPath(),
		oriImgList: req.GetImages(),
	}
}

// Save receives a save request and save the image(s) into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) error {
	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
	}).Info("SaveRequest received")

	opt := b.getSaveOption(req)
	archWriter, err := archive.NewWriter(opt.sysCtx, opt.outputPath)
	if err != nil {
		return errors.Errorf("create archive writer failed: %v", err)
	}
	defer func() {
		if err = archWriter.Close(); err != nil {
			logrus.Errorf("Close archive writer failed: %v", err)
		}
	}()
	opt.writer = archWriter
	opt.images, err = getImagesFromLocal(opt.oriImgList, opt.localStore)
	if err != nil {
		return err
	}

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), opt.saveID)
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(exportHandler(ctx, stream, opt))
	eg.Go(messageHandler(stream, opt.logger))
	errC := make(chan error, 1)

	go func() { errC <- eg.Wait() }()
	defer close(errC)

	select {
	case err2, ok := <-errC:
		if !ok {
			logrus.WithField(util.LogKeySessionID, opt.saveID).Info("Channel errC closed")
			return nil
		}
		if err2 != nil {
			return err2
		}
	case _, ok := <-stream.Context().Done():
		if !ok {
			logrus.WithField(util.LogKeySessionID, opt.saveID).Info("Channel stream done closed")
			return nil
		}
		err = egCtx.Err()
		if err != nil && err != context.Canceled {
			logrus.WithField(util.LogKeySessionID, opt.saveID).Infof("Stream closed with: %v", err)
		}
	}

	return nil
}

func getImagesFromLocal(imageList []string, localStore *store.Store) (map[string]*imageInfo, error) {
	var images = make(map[string]*imageInfo)
	for _, image := range imageList {
		_, localImg, err := im.FindImageLocally(localStore, image)
		if err != nil {
			logrus.Errorf("Lookup local image %s failed: %v", image, err)
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
				return nil, errors.Errorf("invalid tag %s (%s): dose not contain a tag", tag, ref.String())
			}
			imgInfo.tags = append(imgInfo.tags, refTag)
		}
	}

	return images, nil
}

func exportHandler(ctx context.Context, stream pb.Control_SaveServer, options saveOptions) func() error {
	return func() error {
		var finalErr error
		defer func() {
			options.logger.CloseContent()
			if finalErr != nil {
				if rErr := os.Remove(options.outputPath); rErr != nil && !os.IsNotExist(rErr) {
					logrus.Warnf("Removing save output tarball %q failed: %v", options.outputPath, rErr)
				}
			}
		}()

		policyContext, err := exporter.NewPolicyContext(options.sysCtx)
		if err != nil {
			finalErr = err
			logrus.Errorf("Getting policy failed: %v", err)
			return errors.Wrap(err, "error getting policy")
		}
		defer func() {
			if err := policyContext.Destroy(); err != nil {
				logrus.Debugf("Error destroying signature policy context: %v", err)
			}
		}()

		for id, img := range options.images {
			exOpts := exporter.ExportOptions{
				Ctx:           ctx,
				SystemContext: options.sysCtx,
				ExportID:      options.saveID,
				ReportWriter:  options.logger,
			}
			src, err := is.Transport.NewStoreReference(options.localStore, nil, id)
			if err != nil {
				finalErr = err
				logrus.Errorf("Getting source image %s failed: %v", img.oriName, err)
				return errors.Wrapf(err, "error getting source image ref for %q", img.oriName)
			}
			dest, errW := options.writer.NewReference(nil)
			if errW != nil {
				finalErr = errW
				return errW
			}
			copyOptions := exporter.NewCopyOptions(exOpts)
			copyOptions.DestinationCtx.DockerArchiveAdditionalTags = img.tags
			copyOptions.SourceCtx.DockerArchiveAdditionalTags = img.tags
			_, err = cp.Image(stream.Context(), policyContext, dest, src, copyOptions)
			if err != nil {
				finalErr = err
				logrus.Errorf("Copying source image %s failed: %v", img.oriName, err)
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
