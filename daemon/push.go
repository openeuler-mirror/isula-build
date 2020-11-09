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

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/transports/alltransports"
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
	sysCtx     *types.SystemContext
	logger     *logger.Logger
	localStore *store.Store
	pushID     string
	imageName  string
}

// Push receives a push request and push the image to remote repository
func (b *Backend) Push(req *pb.PushRequest, stream pb.Control_PushServer) error {
	logrus.WithFields(logrus.Fields{
		"PushID":    req.GetPushID(),
		"ImageName": req.GetImageName(),
	}).Info("PushRequest received")

	cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)

	opt := pushOptions{
		sysCtx:     image.GetSystemContext(),
		logger:     cliLogger,
		localStore: b.daemon.localStore,
		pushID:     req.GetPushID(),
		imageName:  req.GetImageName(),
	}

	eg, egCtx := errgroup.WithContext(stream.Context())

	eg.Go(pushHandler(egCtx, opt))
	eg.Go(pushMessageHandler(stream, opt.logger))
	errC := make(chan error, 1)

	errC <- eg.Wait()
	defer close(errC)

	err, ok := <-errC
	if !ok {
		logrus.WithField(util.LogKeySessionID, opt.pushID).Info("Channel errC closed")
		return nil
	}
	if err != nil {
		logrus.WithField(util.LogKeySessionID, opt.pushID).Warnf("Stream closed with: %v", err)
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
		}

		policyContext, err := exporter.NewPolicyContext(exOpts.SystemContext)
		if err != nil {
			logrus.Errorf("Getting policy failed: %v", err)
			return errors.Wrap(err, "error getting policy")
		}
		defer func() {
			if pErr := policyContext.Destroy(); pErr != nil {
				logrus.Debugf("Error destroying signature policy context: %v", pErr)
			}
		}()

		src, img, fErr := image.FindImage(options.localStore, options.imageName)
		if fErr != nil {
			return errors.Wrapf(fErr, "find local image %v error", options.imageName)
		}
		dest, pErr := alltransports.ParseImageName(util.DefaultTransport + options.imageName)
		if pErr != nil {
			return errors.Errorf("parse dest spec: %q failed, got error: %v", options.imageName, pErr)
		}
		cpOpt := exporter.NewCopyOptions(exOpts)

		if _, err = cp.Image(exOpts.Ctx, policyContext, dest, src, cpOpt); err != nil {
			logrus.Errorf("Copying source image %s failed: %v", img.Names[0], err)
			return errors.Wrapf(err, "copying source image %s failed", img.Names[0])
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
