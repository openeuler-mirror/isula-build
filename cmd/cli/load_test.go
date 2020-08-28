// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Feiyu Yang
// Create: 2020-07-17
// Description: This file is for image load test.

package main

import (
	"context"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestLoadCmd(t *testing.T) {
	cmd := NewLoadCmd()
	err := cmd.Execute()
	assert.Equal(t, err != nil, true)
	err = loadCommand(cmd, nil)
	assert.ErrorContains(t, err, "isula_build")
}

func TestRunLoad(t *testing.T) {
	ctx := context.Background()
	mockDaemon := newMockDaemon()
	cli := newMockClient(&mockGrpcClient{loadFunc: mockDaemon.load})
	fileEmpty := "empty.tar"
	fileNormal := "test.tar"

	ctxDir := fs.NewDir(t, "load", fs.WithFile(fileEmpty, ""), fs.WithFile(fileNormal, "test"))
	defer ctxDir.Remove()

	type testcase struct {
		name      string
		path      string
		errString string
		isErr     bool
	}
	var testcases = []testcase{
		{
			name:      "path_empty",
			path:      "",
			isErr:     true,
			errString: "not be empty",
		},
		{
			name:      "path_not_exist",
			path:      "not exist",
			isErr:     true,
			errString: "stat",
		},
		{
			name:      "tar_size_zero",
			path:      filepath.Join(ctxDir.Path(), fileEmpty),
			isErr:     true,
			errString: "empty",
		},
		{
			name: "load_normal",
			path: filepath.Join(ctxDir.Path(), fileNormal),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			loadOpts.path = tc.path
			err := runLoad(ctx, &cli)
			assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
			if err != nil {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}
