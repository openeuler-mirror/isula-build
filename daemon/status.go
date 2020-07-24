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
// Create: 2020-01-20
// Description: This file is used to get build status

package daemon

import (
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
)

// status store the key info related to Build action
type status struct {
	// if building start, we notify Status rpc
	startBuild chan struct{}
}

// Status gets build info from backend and send it to the front end
func (b *Backend) Status(req *pb.StatusRequest, stream pb.Control_StatusServer) error {
	logrus.WithFields(logrus.Fields{
		"BuildID": req.GetBuildID(),
	}).Info("StatusRequest received")

	// waiting for Build start first so that the builder with req.BuildID will be set already
	<-b.syncBuildStatus(req.BuildID)

	builder, err := b.daemon.Builder(req.BuildID)
	if err != nil {
		return err
	}
	for value := range builder.StatusChan() {
		if err := stream.Send(&pb.StatusResponse{Content: value}); err != nil {
			return err
		}
	}

	return nil
}

// syncBuildStatus ensure that Build action and Status action can be sync so that to avoid nil point error.
func (b *Backend) syncBuildStatus(buildID string) chan struct{} {
	b.Lock()
	if _, ok := b.status[buildID]; !ok {
		statusPerID := &status{startBuild: make(chan struct{})}
		b.status[buildID] = statusPerID
	}
	statusChan := b.status[buildID].startBuild
	b.Unlock()
	return statusChan
}
