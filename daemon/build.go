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
	"context"
	"io/ioutil"

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

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeyBuildID), req.BuildID)
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
		imageID  string
		pipeFile string
		eg       *errgroup.Group
		fileChan chan []byte
		errc     = make(chan error, 1)
	)

	pipeWrapper := builder.OutputPipeWrapper()
	eg, ctx = errgroup.WithContext(ctx)
	eg.Go(func() error {
		if pipeWrapper != nil {
			pipeFile = pipeWrapper.PipeFile
			defer pipeWrapper.Close()
		}
		b.syncBuildStatus(req.BuildID) <- struct{}{}
		imageID, err = builder.Build()

		// in case there is error during Build stage, the backend will always waiting for content write into
		// the pipeFile, which will cause frontend hangs forever.
		// so if the output type is archive(pipeFile is not empty string) and any error occurred, we write the error
		// message into the pipe to make the goroutine move on instead of hangs.
		if err != nil && pipeFile != "" {
			if perr := ioutil.WriteFile(pipeFile, []byte(err.Error()), constant.DefaultRootFileMode); perr != nil {
				logrus.WithField(util.LogKeyBuildID, req.BuildID).Warnf("Write error [%v] in to pipe file failed: %v", err, perr)
			}
		}

		return err
	})

	eg.Go(func() error {
		if pipeWrapper == nil {
			return nil
		}
		fileChan, err = exporter.PipeArchiveStream(req.BuildID, pipeWrapper)
		if err != nil {
			return err
		}

		for c := range fileChan {
			if err = stream.Send(&pb.BuildResponse{
				Data: c,
			}); err != nil {
				return err
			}
		}
		return pipeWrapper.Err
	})

	go func() {
		errc <- eg.Wait()
	}()

	select {
	case err = <-errc:
		close(errc)
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
			logrus.WithField(util.LogKeyBuildID, req.BuildID).Warnf("Stream closed with: %v", err)
		}
	}

	return nil
}
