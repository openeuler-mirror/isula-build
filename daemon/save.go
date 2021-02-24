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

type saveOptions struct {
	sysCtx     *types.SystemContext
	logger     *logger.Logger
	localStore *store.Store
	logEntry   *logrus.Entry
	saveID     string
	outputPath string
	oriImgList []string
	format     string
}

func (b *Backend) getSaveOptions(req *pb.SaveRequest) saveOptions {
	return saveOptions{
		sysCtx:     image.GetSystemContext(),
		logger:     logger.NewCliLogger(constant.CliLogBufferLen),
		localStore: b.daemon.localStore,
		saveID:     req.GetSaveID(),
		outputPath: req.GetPath(),
		oriImgList: req.GetImages(),
		format:     req.GetFormat(),
		logEntry:   logrus.WithFields(logrus.Fields{"SaveID": req.GetSaveID(), "Format": req.GetFormat()}),
	}
}

// Save receives a save request and save the image(s) into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) error {
	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
		"Format": req.GetFormat(),
	}).Info("SaveRequest received")

	var (
		ok  bool
		err error
	)

	opts := b.getSaveOptions(req)

	switch opts.format {
	case exporter.DockerTransport:
		opts.format = exporter.DockerArchiveTransport
	case exporter.OCITransport:
		opts.format = exporter.OCIArchiveTransport
	default:
		return errors.New("wrong image format provided")
	}

	for i, imageName := range opts.oriImgList {
		nameWithTag, cErr := image.CheckAndAddDefaultTag(imageName, opts.localStore)
		if cErr != nil {
			return cErr
		}
		opts.oriImgList[i] = nameWithTag
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

	eg.Go(exportHandler(ctx, opts))
	eg.Go(messageHandler(stream, opts.logger))
	errC := make(chan error, 1)

	errC <- eg.Wait()
	defer close(errC)

	err, ok = <-errC
	if !ok {
		opts.logEntry.Info("Channel errC closed")
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

func exportHandler(ctx context.Context, opts saveOptions) func() error {
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

		for _, imageID := range opts.oriImgList {
			exOpts := exporter.ExportOptions{
				Ctx:           ctx,
				SystemContext: opts.sysCtx,
				ExportID:      opts.saveID,
				ReportWriter:  opts.logger,
			}

			if err := exporter.Export(imageID, exporter.FormatTransport(opts.format, opts.outputPath),
				exOpts, opts.localStore); err != nil {
				opts.logEntry.Errorf("Save Image %s output to %s failed with: %v", imageID, opts.format, err)
				return errors.Wrapf(err, "save Image %s output to %s failed", imageID, opts.format)
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
