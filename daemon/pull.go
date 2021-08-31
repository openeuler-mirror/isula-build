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
// Description: This file is "pull" command for backend

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
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type pullOptions struct {
	sysCtx     *types.SystemContext
	logger     *logger.Logger
	localStore *store.Store
	pullID     string
	imageName  string
}

// Pull receives a pull request and pull the image from remote repository
func (b *Backend) Pull(req *pb.PullRequest, stream pb.Control_PullServer) error {
	logrus.WithFields(logrus.Fields{
		"PullID":    req.GetPullID(),
		"ImageName": req.GetImageName(),
	}).Info("PullRequest received")

	cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)
	opt := pullOptions{
		sysCtx:     image.GetSystemContext(),
		logger:     cliLogger,
		localStore: b.daemon.localStore,
		pullID:     req.GetPullID(),
		imageName:  req.GetImageName(),
	}

	if _, err := dockerref.Parse(opt.imageName); err != nil {
		return err
	}

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), req.GetPullID())
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(pullHandler(egCtx, opt))
	eg.Go(pullMessageHandler(stream, opt.logger))

	if err := eg.Wait(); err != nil {
		logrus.WithField(util.LogKeySessionID, opt.pullID).Warnf("Pull stream closed with: %v", err)
		return err
	}

	return nil
}

func pullHandler(ctx context.Context, options pullOptions) func() error {
	return func() error {
		defer func() {
			options.logger.CloseContent()
		}()

		if _, _, err := image.PullAndGetImageInfo(&image.PrepareImageOptions{
			Ctx:           ctx,
			FromImage:     options.imageName,
			SystemContext: options.sysCtx,
			Store:         options.localStore,
			Reporter:      options.logger,
		}); err != nil {
			return errors.Wrapf(err, "copying source image %s failed", options.imageName)
		}

		return nil
	}
}

func pullMessageHandler(stream pb.Control_PullServer, cliLogger *logger.Logger) func() error {
	return func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err := stream.Send(&pb.PullResponse{
				Response: content,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}
