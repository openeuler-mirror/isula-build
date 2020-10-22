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
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type saveOptions struct {
	store     *store.Store
	imageName string
	imageID   string
	saveID    string
	output    string
	imageInfo string
}

// Save receives a save request and save the image into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) error {
	var (
		opts *saveOptions
		err  error
	)

	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
	}).Info("SaveRequest received")

	if opts, err = b.preSave(req); err != nil {
		return err
	}

	egCtx, errC := b.doSave(req, stream, opts)
	defer close(errC)
	select {
	case err2, ok := <-errC:
		if !ok {
			logrus.WithField(util.LogKeySessionID, req.GetSaveID()).Info("Channel errC closed")
			return nil
		}
		if err2 != nil {
			return err2
		}
	case _, ok := <-stream.Context().Done():
		if !ok {
			logrus.WithField(util.LogKeySessionID, req.GetSaveID()).Info("Channel stream done closed")
			return nil
		}
		err = egCtx.Err()
		if err != nil && err != context.Canceled {
			logrus.WithField(util.LogKeySessionID, req.GetSaveID()).Infof("Stream closed with: %v", err)
		}
	}

	return nil
}

func (b *Backend) preSave(req *pb.SaveRequest) (*saveOptions, error) {
	const exportType = "docker-archive"
	var (
		imageName  string
		err        error
		localStore = b.daemon.localStore
	)

	_, img, err := image.FindImage(localStore, req.GetImage())
	if err != nil {
		logrus.Errorf("Lookup image %s failed: %v", req.GetImage(), err)
		return nil, err
	}

	output := fmt.Sprintf("%s:%s", exportType, req.GetPath())
	imageID := img.ID
	imageName, err = checkTag(req.GetImage(), imageID)
	if err != nil {
		return nil, err
	}
	// if image has tag with it
	if imageName != "" {
		output = fmt.Sprintf("%s:%s:%s", exportType, req.GetPath(), imageName)
	}

	opts := &saveOptions{
		imageName: imageName,
		imageID:   imageID,
		saveID:    req.GetSaveID(),
		store:     localStore,
		output:    output,
		imageInfo: req.GetImage(),
	}
	return opts, nil
}

func (b *Backend) doSave(req *pb.SaveRequest, stream pb.Control_SaveServer, opt *saveOptions) (context.Context, chan error) {
	var (
		errC      = make(chan error, 1)
		cliLogger = logger.NewCliLogger(constant.CliLogBufferLen)
	)

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), req.GetSaveID())
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(exportHandler(ctx, opt, cliLogger))
	eg.Go(messageHandler(stream, cliLogger))

	go func() {
		errC <- eg.Wait()
	}()

	return egCtx, errC
}

func exportHandler(ctx context.Context, opts *saveOptions, cliLogger *logger.Logger) func() error {
	return func() error {
		defer func() {
			cliLogger.CloseContent()
		}()
		exOpts := exporter.ExportOptions{
			SystemContext: image.GetSystemContext(),
			Ctx:           ctx,
			ReportWriter:  cliLogger,
			ExportID:      opts.saveID,
		}

		if err := exporter.Export(opts.imageID, opts.output, exOpts, opts.store); err != nil {
			logrus.Errorf("Save image %s failed: %v", opts.imageInfo, err)
			return err
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

func checkTag(oriImg, imageID string) (string, error) {
	// no tag found
	if strings.HasPrefix(imageID, oriImg) {
		return "", nil
	}
	tag, err := dockerfile.CheckAndExpandTag(oriImg)
	if err != nil {
		return "", err
	}
	return tag, nil
}
