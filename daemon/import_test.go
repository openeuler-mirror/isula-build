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
// Create: 2022-12-01
// Description: This file tests import interface.

package daemon

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type controlImportServer struct {
	grpc.ServerStream
}

func (c *controlImportServer) Send(response *pb.ImportResponse) error {
	if response.Log == "error" {
		return errors.New("error happened")
	}
	return nil
}

func (c *controlImportServer) Context() context.Context {
	return context.Background()
}

func TestImport(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	tmpFile := fs.NewFile(t, t.Name())
	defer tmpFile.Remove()
	err := ioutil.WriteFile(tmpFile.Path(), []byte("This is test file"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	importID := util.GenerateNonCryptoID()[:constant.DefaultIDLen]

	testcases := []struct {
		name      string
		req       *pb.ImportRequest
		wantErr   bool
		errString string
	}{
		{
			name: "TC1 - normal case",
			req: &pb.ImportRequest{
				ImportID:  importID,
				Source:    tmpFile.Path(),
				Reference: "test:image",
			},
			wantErr:   true,
			errString: "Error processing tar file",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &controlImportServer{}
			err := d.Daemon.backend.Import(tc.req, stream)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}

}
