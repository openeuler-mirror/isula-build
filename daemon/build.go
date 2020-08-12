// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-01-20
// Description: This file is "build" command for backend

package daemon

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/util"
)

// Build receives a build request and build an image
func (b *Backend) Build(req *pb.BuildRequest, stream pb.Control_BuildServer) (err error) { // nolint:gocyclo
	logrus.WithFields(logrus.Fields{
		"BuildType": req.GetBuildType(),
		"BuildID":   req.GetBuildID(),
	}).Info("BuildRequest received")

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), req.BuildID)
	builder, err := b.daemon.NewBuilder(ctx, req)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := builder.CleanResources(); cerr != nil {
			logrus.Warnf("defer builder clean build resources failed: %v", cerr)
		}
		b.daemon.deleteBuilder(req.BuildID)
		b.deleteStatus(req.BuildID)
	}()

	var (
		f       *os.File
		length  int
		imageID string
		eg      *errgroup.Group
		errC    = make(chan error, 1)
	)

	pipeWrapper := builder.OutputPipeWrapper()
	eg, ctx = errgroup.WithContext(ctx)
	eg.Go(func() error {
		b.syncBuildStatus(req.BuildID) <- struct{}{}
		imageID, err = builder.Build()

		// in case there is error during Build stage, the backend will always waiting for content write into
		// the pipeFile, which will cause frontend hangs forever.
		// so if the output type is archive(pipeFile is not empty string) and any error occurred, we write the error
		// message into the pipe to make the goroutine move on instead of hangs.
		if err != nil && pipeWrapper != nil {
			pipeWrapper.Close()
			if perr := ioutil.WriteFile(pipeWrapper.PipeFile, []byte(err.Error()), constant.DefaultRootFileMode); perr != nil {
				logrus.WithField(util.LogKeySessionID, req.BuildID).Warnf("Write error [%v] in to pipe file failed: %v", err, perr)
			}
		}

		return err
	})

	eg.Go(func() error {
		if pipeWrapper == nil {
			return nil
		}
		f, err = exporter.PipeArchiveStream(pipeWrapper)
		defer func() {
			if cErr := f.Close(); cErr != nil {
				logrus.WithField(util.LogKeySessionID, req.BuildID).Warnf("Closing archive stream pipe %q failed: %v", pipeWrapper.PipeFile, cErr)
			}
		}()
		if err != nil {
			return err
		}

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
			if err = stream.Send(&pb.BuildResponse{
				Data: buf[0:length],
			}); err != nil {
				return err
			}
		}
		logrus.WithField(util.LogKeySessionID, req.BuildID).Debugf("Piping build archive stream done")
		return nil
	})

	go func() {
		errC <- eg.Wait()
	}()

	select {
	case err = <-errC:
		close(errC)
		if err != nil {
			return err
		}
		// export done, send client the imageID
		if err = stream.Send(&pb.BuildResponse{
			Data:    nil,
			ImageID: imageID,
		}); err != nil {
			return err
		}
	case <-stream.Context().Done():
		err = ctx.Err()
		if err != nil && err != context.Canceled {
			logrus.WithField(util.LogKeySessionID, req.BuildID).Warnf("Stream closed with: %v", err)
		}
	}

	return nil
}
