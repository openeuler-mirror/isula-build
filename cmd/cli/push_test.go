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
// Description: This file is used for testing command push

package main

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestPushCommand(t *testing.T) {
	testcases := []struct {
		name      string
		args      []string
		format    string
		wantErr   bool
		errString string
	}{
		{
			name:      "normal case with image format docker",
			args:      []string{"openeuler:latest"},
			format:    "docker",
			wantErr:   true,
			errString: "isula_build.sock",
		},
		{
			name:      "abnormal case with multiple args",
			args:      []string{"aaa", "bbb"},
			format:    "oci",
			wantErr:   true,
			errString: "push requires exactly one argument",
		},
		{
			name:      "abnormal case with empty args",
			args:      []string{""},
			format:    "docker",
			wantErr:   true,
			errString: "repository name must have at least one component",
		},
		{
			name:      "abnormal case with invalid args",
			args:      []string{"busybox-:latest"},
			format:    "oci",
			wantErr:   true,
			errString: "invalid reference format",
		},
		{
			name:      "normal case with image format oci",
			args:      []string{"openeuler:latest"},
			format:    "oci",
			wantErr:   true,
			errString: "isula_build.sock",
		},
		{
			name:      "abnormal case with wrong image format dock",
			args:      []string{"openeuler:latest"},
			format:    "dock",
			wantErr:   true,
			errString: "wrong image format",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			{
				pushCmd := NewPushCmd()
				pushOpts.format = tc.format

				err := pushCommand(pushCmd, tc.args)
				if tc.wantErr {
					assert.ErrorContains(t, err, tc.errString)
				}
			}
		})

	}
}

func TestRunPush(t *testing.T) {
	ctx := context.Background()
	mockPush := newMockDaemon()
	cli := newMockClient(&mockGrpcClient{pushFunc: mockPush.push})

	testcases := []struct {
		name      string
		imageName string
		wantErr   bool
		errString string
	}{
		{
			name:      "normal case",
			imageName: "registry.example.com/library/image:test",
			wantErr:   false,
		},
		{
			name:      "abnormal case with wrong image format",
			imageName: "registry.example.com/library/image-:test",
			wantErr:   true,
			errString: "invalid format of image name",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := runPush(ctx, &cli, tc.imageName)
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}
