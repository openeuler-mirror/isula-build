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
// Create: 2022-12-02
// Description: This file tests remove interface.

package daemon

import (
	"context"
	"testing"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
)

type controlRemoveServer struct {
	grpc.ServerStream
}

func (c *controlRemoveServer) Send(response *pb.RemoveResponse) error {
	if response.LayerMessage == "error" {
		return errors.New("error happened")
	}
	return nil
}

func (c *controlRemoveServer) Context() context.Context {
	return context.Background()
}

func TestRemove(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	options := &storage.ImageOptions{}
	testImg := make([]string, 0)
	testImg1, err := d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"test:image1"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	d.Daemon.localStore.SetNames(testImg1.ID, append(testImg1.Names, "test:image1-backup"))
	testImg = append(testImg, testImg1.ID)
	testImg2, err := d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"test:image2"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	testImg = append(testImg, testImg2.ID)

	testcases := []struct {
		name      string
		req       *pb.RemoveRequest
		wantErr   bool
		errString string
	}{
		{
			name: "TC1 - normal case",
			req: &pb.RemoveRequest{
				ImageID: testImg,
				All:     true,
				Prune:   false,
			},
			wantErr: false,
		},
		{
			name: "TC2 - abnormal case with no images",
			req: &pb.RemoveRequest{
				ImageID: []string{""},
				All:     false,
				Prune:   false,
			},
			wantErr:   true,
			errString: "remove one or more images failed",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &controlRemoveServer{}
			err := d.Daemon.backend.Remove(tc.req, stream)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}
}
