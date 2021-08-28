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
// Create: 2021-02-03
// Description: This file tests List interface

package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/bndr/gotabulate"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/stringid"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
)

func TestList(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	options := &storage.ImageOptions{}
	img, err := d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image:test1"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image:test2"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"egami:test"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	// image with no name and tag
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	d.Daemon.localStore.SetNames(img.ID, append(img.Names, "image:test1-backup"))
	// image who's repo contains port
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"hub.example.com:8080/image:test"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}

	testcases := []struct {
		name      string
		req       *pb.ListRequest
		wantErr   bool
		errString string
	}{
		{
			name: "normal case list specific image with repository[:tag]",
			req: &pb.ListRequest{
				ImageName: "image:test1",
			},
			wantErr: false,
		},
		{
			name: "normal case list specific image with image id",
			req: &pb.ListRequest{
				ImageName: img.ID,
			},
			wantErr: false,
		},
		{
			name: "normal case list all images",
			req: &pb.ListRequest{
				ImageName: "",
			},
			wantErr: false,
		},
		{
			name: "normal case list all images with repository",
			req: &pb.ListRequest{
				ImageName: "image",
			},
			wantErr: false,
		},
		{
			name: "abnormal case no image found in local store",
			req: &pb.ListRequest{
				ImageName: "coffee:costa",
			},
			wantErr:   true,
			errString: "not found in local store",
		},
		{
			name: "abnormal case no repository",
			req: &pb.ListRequest{
				ImageName: "coffee",
			},
			wantErr:   true,
			errString: "failed to list images with repository",
		},
		{
			name: "abnormal case ImageName only contains latest tag",
			req: &pb.ListRequest{
				ImageName: ":latest",
			},
			wantErr:   true,
			errString: "invalid reference format",
		},
		{
			name: "normal case ImageName contains port number and tag",
			req: &pb.ListRequest{
				ImageName: "hub.example.com:8080/image:test",
			},
			wantErr: false,
		},
		{
			name: "normal case ImageName contains port number",
			req: &pb.ListRequest{
				ImageName: "hub.example.com:8080/image",
			},
			wantErr: false,
		},
		{
			name: "abnormal case wrong ImageName",
			req: &pb.ListRequest{
				ImageName: "hub.example.com:8080/",
			},
			wantErr:   true,
			errString: "failed to list images with repository",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			resp, err := d.Daemon.backend.List(ctx, tc.req)

			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
				formatAndPrint(resp.Images)
			}
		})
	}
}

func formatAndPrint(images []*pb.ListResponse_ImageInfo) {
	emptyStr := `-----------   ----   ---------   --------
	REPOSITORY    TAG    IMAGE ID    CREATED
	-----------   ----   ---------   --------`
	lines := make([][]string, 0, len(images))
	title := []string{"REPOSITORY", "TAG", "IMAGE ID", "CREATED", "SIZE"}
	for _, image := range images {
		if image == nil {
			continue
		}
		line := []string{image.Repository, image.Tag, image.Id[:constant.DefaultIDLen], image.Created, image.Size_}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		fmt.Println(emptyStr)
		return
	}
	tabulate := gotabulate.Create(lines)
	tabulate.SetHeaders(title)
	tabulate.SetAlign("left")
	fmt.Print(tabulate.Render("simple"))
}
