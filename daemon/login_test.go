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
// Description: This file tests login interface.

package daemon

import (
	"context"
	"crypto/sha512"
	"testing"

	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

func TestLogin(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	encryptKey, err := util.EncryptRSA("testpassword", d.Daemon.backend.daemon.key.PublicKey, sha512.New())
	assert.NilError(t, err)
	testcases := []struct {
		name      string
		req       *pb.LoginRequest
		wantErr   bool
		errString string
	}{
		{
			name: "TC1 - normal case with abnormal password",
			req: &pb.LoginRequest{
				Server:   "testcase.org",
				Username: "testuser",
				Password: "decfabdc",
			},
			wantErr:   true,
			errString: "decryption failed",
		},
		{
			name: "TC2 - normal case with abnormal registry",
			req: &pb.LoginRequest{
				Server:   "testcase.org",
				Username: "testuser",
				Password: encryptKey,
			},
			wantErr:   true,
			errString: "no route to host",
		},
		{
			name: "TC3 - abnormal case with empty server",
			req: &pb.LoginRequest{
				Server:   "",
				Username: "testuser",
				Password: "testpassword",
			},
			wantErr:   true,
			errString: "empty server address",
		},
		{
			name: "TC4 - abnormal case with empty password",
			req: &pb.LoginRequest{
				Server:   "test.org",
				Username: "testuser",
				Password: "",
			},
			wantErr:   true,
			errString: "empty auth info",
		},
		{
			name: "TC5 - abnormal case with empty password and username",
			req: &pb.LoginRequest{
				Server:   "test.org",
				Username: "",
				Password: "",
			},
			wantErr:   true,
			errString: "failed to read auth file",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			_, err := d.Daemon.backend.Login(ctx, tc.req)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}
}
