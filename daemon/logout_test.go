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
// Create: 2022-11-29
// Description: This file tests logout interface.

package daemon

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
)

func TestLogout(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	testcases := []struct {
		name      string
		req       *pb.LogoutRequest
		wantErr   bool
		errString string
	}{
		{
			name: "TC1 - normal case",
			req: &pb.LogoutRequest{
				Server: "test.org",
				All:    true,
			},
			wantErr: false,
		},
		{
			name: "TC2 - abnormal case with empty server",
			req: &pb.LogoutRequest{
				Server: "",
				All:    false,
			},
			wantErr:   true,
			errString: "empty server address",
		},
		{
			name: "TC3 - abnormal case with no logined registry",
			req: &pb.LogoutRequest{
				Server: "test.org",
				All:    false,
			},
			wantErr:   true,
			errString: "not logged in",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			_, err := d.Daemon.backend.Logout(ctx, tc.req)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}
}
