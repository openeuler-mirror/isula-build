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
// Description: This file is used for testing command pull

package main

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestPullCommand(t *testing.T) {
	testcases := []struct {
		name      string
		args      []string
		cli       Cli
		wantErr   bool
		errString string
	}{
		{
			name:      "normal case",
			args:      []string{"openeuler:latest"},
			wantErr:   true,
			errString: "isula_build.sock",
		},
		{
			name:      "abnormal case with multiple args",
			args:      []string{"aaa", "bbb"},
			wantErr:   true,
			errString: "pull requires exactly one argument",
		},
		{
			name:      "abnormal case with empty args",
			args:      []string{""},
			wantErr:   true,
			errString: "repository name must have at least one component",
		},
		{
			name:      "abnormal case with invalid args",
			args:      []string{"busybox-:latest"},
			wantErr:   true,
			errString: "invalid reference format",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pullCmd := NewPullCmd()
			err := pullCommand(pullCmd, tc.args)
			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}

func TestRunPull(t *testing.T) {
	ctx := context.Background()
	mockPull := newMockDaemon()
	cli := newMockClient(&mockGrpcClient{pullFunc: mockPull.pull})

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
			err := runPull(ctx, &cli, tc.imageName)
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}
