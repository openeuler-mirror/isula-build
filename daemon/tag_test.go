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
// Create: 2022-12-04
// Description: This file tests tag interface.

package daemon

import (
	"context"
	"testing"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/stringid"
	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
)

func TestTag(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	options := &storage.ImageOptions{}
	_, err := d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"test:image"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	testcases := []struct {
		name      string
		req       *pb.TagRequest
		wantErr   bool
		errString string
	}{
		{
			name: "TC1 - normal case",
			req: &pb.TagRequest{
				Image: "test:image",
				Tag:   "image-backup",
			},
			wantErr: false,
		},
		{
			name: "TC2 - abnormal case with no existed images",
			req: &pb.TagRequest{
				Image: "",
				Tag:   "image-backup",
			},
			wantErr:   true,
			errString: "repository name must have at least one component",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			_, err := d.Daemon.backend.Tag(ctx, tc.req)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}
}
