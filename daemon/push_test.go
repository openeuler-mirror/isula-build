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

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gotest.tools/assert"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/pkg/logger"
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
	stream := &controlPushServer{}

	testCases := []struct {
		testName    string
		pushRequest *pb.PushRequest
	}{
		{
			testName: "localNotExist",
			pushRequest: &pb.PushRequest{
				PushID:    stringid.GenerateNonCryptoID()[:constant.DefaultIDLen],
				ImageName: "255.255.255.255/no-repository/no-name",
			},
		},
		{
			testName: "manifestNotExist",
			pushRequest: &pb.PushRequest{
				PushID:    stringid.GenerateNonCryptoID()[:constant.DefaultIDLen],
				ImageName: "127.0.0.1/no-repository/no-name:latest",
			},
		},
	}

	options := &storage.ImageOptions{}
	d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"127.0.0.1/no-repository/no-name:latest"}, "", "", options)

	for _, tc := range testCases {
		err := d.Daemon.backend.Push(tc.pushRequest, stream)
		if tc.testName == "localNotExist" {
			assert.ErrorContains(t, err, "error: locating image")
		}
		if tc.testName == "manifestNotExist" {
			assert.ErrorContains(t, err, "file does not exist")
		}
	}

	tmpClean(d)
}

func TestPushHandler(t *testing.T) {
	stream := &controlPushServer{}
	cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)

	ctx := context.TODO()
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(pushMessageHandler(stream, cliLogger))
	eg.Go(func() error {
		cliLogger.Print("Push Response")
		cliLogger.CloseContent()
		return nil
	})

	eg.Wait()
}
