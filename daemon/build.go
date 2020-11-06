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

	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

// Build receives a build request and build an image
func (b *Backend) Build(ctx context.Context, req *pb.BuildRequest) (*pb.BuildResponse, error) {
	b.wg.Add(1)
	defer b.wg.Done()
	logEntry := logrus.WithFields(logrus.Fields{"BuildType": req.GetBuildType(), "BuildID": req.GetBuildID()})
	logEntry.Info("BuildRequest received")

	ctx = context.WithValue(ctx, util.LogFieldKey(util.LogKeySessionID), req.BuildID)
	builder, nErr := b.daemon.NewBuilder(ctx, req)
	if nErr != nil {
		logEntry.Error(nErr)
		return &pb.BuildResponse{}, nErr
	}

	defer func() {
		if cErr := builder.CleanResources(); cErr != nil {
			logEntry.Warnf("defer builder clean build resources failed: %v", cErr)
		}
		b.daemon.deleteBuilder(req.BuildID)
		b.deleteStatus(req.BuildID)
	}()

	b.syncBuildStatus(req.BuildID) <- struct{}{}
	b.closeStatusChan(req.BuildID)
	imageID, bErr := builder.Build()
	if bErr != nil {
		logEntry.Error(bErr)
		return &pb.BuildResponse{}, bErr
	}

	return &pb.BuildResponse{ImageID: imageID}, nil
}
