// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: daisicheng
// Create: 2022-12-09
// Description: This file tests status interface.

package daemon

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type controlStatusServer struct {
	grpc.ServerStream
}

func (c *controlStatusServer) Send(response *pb.StatusResponse) error {
	if response.Content == "error" {
		return errors.New("error happened")
	}
	return nil
}

func (c *controlStatusServer) Context() context.Context {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	return ctx
}

func TestStatus(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	buildID := util.GenerateNonCryptoID()[:constant.DefaultIDLen]
	stream := &controlStatusServer{}
	err := d.Daemon.backend.Status(&pb.StatusRequest{BuildID: buildID}, stream)
	assert.NilError(t, err)
}
