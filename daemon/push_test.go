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
// Description: This file tests Push interface

package daemon

import (
	"context"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
	"google.golang.org/grpc"
	"gotest.tools/assert"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
)

type controlPushServer struct {
	grpc.ServerStream
}

func (c *controlPushServer) Context() context.Context {
	return context.Background()
}

func (c *controlPushServer) Send(response *pb.PushResponse) error {
	return nil
}

func init() {
	reexec.Init()

}

func TestPush(t *testing.T) {
	d := prepare(t)
	pushID := stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]
	req := &pb.PushRequest{
		PushID:    pushID,
		ImageName: "255.255.255.255/no-repository/no-name",
	}
	stream := &controlPushServer{}
	err := d.Daemon.backend.Push(req, stream)
	assert.ErrorContains(t, err, "error: locating image")
	tmpClean(d)
}
