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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

// Save receives a save request and save the image into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) (err error) { // nolint:gocyclo
	const exportType = "docker-archive"
	var (
		pipeWrapper *exporter.PipeWrapper
		imageInfo   = req.Image
		saveID      = req.SaveID
		store       = b.daemon.localStore
		errC        = make(chan error, 1)
		runDir      = filepath.Join(b.daemon.opts.RunRoot, "save", saveID)
		cliLogger   = logger.NewCliLogger(constant.CliLogBufferLen)
	)

	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
	}).Info("SaveRequest received")

	if err = os.MkdirAll(runDir, constant.DefaultRootDirMode); err != nil {
		return err
	}
	defer func() {
		if rErr := os.RemoveAll(runDir); rErr != nil {
			logrus.Errorf("Remove saving dir %q failed: %v", runDir, rErr)
			err = rErr
		}
	}()

	imageID, err := store.Lookup(imageInfo)
	if err != nil {
		logrus.Errorf("Lookup image %s failed: %v", imageInfo, err)
		return err
	}

	pipeWrapper, err = exporter.NewPipeWrapper(runDir, exportType)
	if err != nil {
		return err
	}

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), saveID)
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer func() {
			cliLogger.CloseContent()
		}()
		output := fmt.Sprintf("%s:%s", exportType, pipeWrapper.PipeFile)
		exOpts := exporter.ExportOptions{
			SystemContext: image.GetSystemContext(),
			Ctx:           ctx,
			ReportWriter:  cliLogger,
		}

		if err = exporter.Export(imageID, output, exOpts, store); err != nil {
			pipeWrapper.Close()
			logrus.Errorf("Save image %s failed: %v", imageID, err)
			return err
		}

		return nil
	})

	eg.Go(func() error {
		var (
			f      *os.File
			length int
		)
		if f, err = exporter.PipeArchiveStream(pipeWrapper); err != nil {
			return err
		}
		defer func() {
			if cErr := f.Close(); cErr != nil {
				logrus.WithField(util.LogKeySessionID, req.SaveID).Warnf("Closing save archive stream pipe %q failed: %v", pipeWrapper.PipeFile, cErr)
			}
		}()

		reader := bufio.NewReader(f)
		buf := make([]byte, constant.BufferSize, constant.BufferSize)
		for {
			length, err = reader.Read(buf)
			if err == io.EOF || pipeWrapper.Done {
				break
			}
			if err != nil {
				return err
			}
			if err = stream.Send(&pb.SaveResponse{
				Data: buf[0:length],
			}); err != nil {
				return err
			}
		}
		logrus.WithField(util.LogKeySessionID, req.SaveID).Debugf("Piping save archive stream done")
		return nil
	})

	eg.Go(func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err = stream.Send(&pb.SaveResponse{
				Log: content,
			}); err != nil {
				return err
			}
		}
		return nil
	})

	go func() {
		errC <- eg.Wait()
	}()

	var ok bool

	select {
	case err, ok = <-errC:
		if !ok {
			logrus.WithField(util.LogKeySessionID, saveID).Warn("Channel errC closed")
			return nil
		}
		close(errC)
		if err != nil {
			return err
		}
		// export done in another go routine, so send nil data
		if err = stream.Send(&pb.SaveResponse{Data: nil}); err != nil {
			return err
		}
	case _, ok = <-stream.Context().Done():
		if !ok {
			logrus.WithField(util.LogKeySessionID, saveID).Warn("Channel stream done closed")
			return nil
		}
		err = egCtx.Err()
		if err != nil && err != context.Canceled {
			logrus.WithField(util.LogKeySessionID, saveID).Warnf("Stream closed with: %v", err)
		}
	}

	return nil
}
