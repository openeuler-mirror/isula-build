// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-08-11
// Description: This file is used for testing command save

package main

import (
	"context"
	"fmt"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestSaveCommand(t *testing.T) {
	saveCmd := NewSaveCmd()
	var args []string
	err := saveCommand(saveCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestRunSave(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	type testcase struct {
		name      string
		path      string
		errString string
		args      []string
		wantErr   bool
	}

	var testcases = []testcase{
		{
			name:    "TC1 - normal case",
			path:    tmpDir.Join("test1"),
			args:    []string{"testImage"},
			wantErr: false,
		},
		{
			name:    "TC2 - abnormal case with empty args",
			path:    tmpDir.Join("test1"),
			args:    []string{},
			wantErr: true,
		},
		{
			name: "TC3 - normal case with relative path",
			path: fmt.Sprintf("./%s", tmpDir.Path()),
			args: []string{"testImage"},
			// err here because cli will try to remove saved tar and it found there is no file to remove
			wantErr:   true,
			errString: "save image failed",
		},
		{
			name:    "TC4 - abnormal case with empty path",
			path:    "",
			args:    []string{"testImage"},
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		ctx := context.Background()
		mockSave := newMockDaemon()
		cli := newMockClient(&mockGrpcClient{saveFunc: mockSave.save})
		saveOpts.path = tc.path
		err := runSave(ctx, &cli, tc.args)
		assert.Equal(t, err != nil, tc.wantErr, "Failed at [%s], err: %v", tc.name, err)
		if err != nil {
			assert.ErrorContains(t, err, tc.errString)
		}
	}
}
