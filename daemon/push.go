// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2020-11-02
// Description: This file is "push" command for backend

package daemon

import (
	"context"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type pushOptions struct {
	sysCtx       *types.SystemContext
	logger       *logger.Logger
	localStore   *store.Store
	pushID       string
	imageName    string
	format       string
	manifestType string
}

// Push receives a push request and push the image to remote repository
func (b *Backend) Push(req *pb.PushRequest, stream pb.Control_PushServer) error {
	logrus.WithFields(logrus.Fields{
		"PushID":    req.GetPushID(),
		"ImageName": req.GetImageName(),
		"Format":    req.GetFormat(),
	}).Info("PushRequest received")

	cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)

	opt := pushOptions{
		sysCtx:     image.GetSystemContext(),
		logger:     cliLogger,
		localStore: b.daemon.localStore,
		pushID:     req.GetPushID(),
		imageName:  req.GetImageName(),
		format:     req.GetFormat(),
	}

	if err := util.CheckImageFormat(opt.format); err != nil {
		return err
	}

	if _, err := dockerref.Parse(opt.imageName); err != nil {
		return err
	}

	manifestType, gErr := exporter.GetManifestType(opt.format)
	if gErr != nil {
		return gErr
	}
	opt.manifestType = manifestType

	eg, egCtx := errgroup.WithContext(stream.Context())

	eg.Go(pushHandler(egCtx, opt))
	eg.Go(pushMessageHandler(stream, opt.logger))

	if err := eg.Wait(); err != nil {
		logrus.WithField(util.LogKeySessionID, opt.pushID).Warnf("Push stream closed with: %v", err)
		return err
	}

	return nil
}

func pushHandler(ctx context.Context, options pushOptions) func() error {
	return func() error {
		defer func() {
			options.logger.CloseContent()
		}()

		exOpts := exporter.ExportOptions{
			Ctx:           ctx,
			SystemContext: options.sysCtx,
			ReportWriter:  options.logger,
			ExportID:      options.pushID,
			ManifestType:  options.manifestType,
		}

		if err := exporter.Export(options.imageName, exporter.FormatTransport(constant.DockerTransport, options.imageName),
			exOpts, options.localStore); err != nil {
			logrus.WithField(util.LogKeySessionID, options.pushID).
				Errorf("Push image %q of format %q failed with %v", options.imageName, constant.DockerTransport, err)
			return errors.Wrapf(err, "push image %q of format %q failed", options.imageName, constant.DockerTransport)
		}

		return nil
	}
}

func pushMessageHandler(stream pb.Control_PushServer, cliLogger *logger.Logger) func() error {
	return func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err := stream.Send(&pb.PushResponse{
				Response: content,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}
