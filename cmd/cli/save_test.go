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

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestSaveCommand(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	alreadyExistFile := fs.NewFile(t, tmpDir.Join("alreadyExist.tar"))
	defer alreadyExistFile.Remove()

	type testcase struct {
		name      string
		path      string
		errString string
		args      []string
		format    string
		wantErr   bool
	}

	// For normal cases, default err is "invalid socket path: unix:///var/run/isula_build.sock". 
	// As daemon is not running as we run unit test.
	var testcases = []testcase{
		{
			name:      "TC1 - normal case with format docker",
			path:      tmpDir.Join("test1"),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "isula_build.sock",
			format:    "docker",
		},
		{
			name:      "TC2 - normal case with format oci",
			path:      tmpDir.Join("test2"),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "isula_build.sock",
			format:    "oci",
		},
		{
			name:      "TC3 - abnormal case path with wrong format",
			path:      tmpDir.Join("test3"),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "wrong image format",
			format:    "dock",
		},
		{
			name:      "TC4 - abnormal case with empty args",
			path:      tmpDir.Join("test4"),
			args:      []string{},
			wantErr:   true,
			errString: "save accepts at least one image",
			format:    "docker",
		},
		{
			name:      "TC5 - normal case with relative path",
			path:      fmt.Sprintf("./%s", tmpDir.Path()),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "isula_build.sock",
			format:    "oci",
		},
		{
			name:      "TC6 - abnormal case with empty path",
			path:      "",
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "output path should not be empty",
			format:    "docker",
		},
		{
			name:      "TC7 - abnormal case with already file exist",
			path:      alreadyExistFile.Path(),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "output file already exist",
			format:    "docker",
		},
		{
			name:      "TC8 - abnormal case path with colon",
			path:      tmpDir.Join("test8:image:tag"),
			args:      []string{"testImage"},
			wantErr:   true,
			errString: "colon in path",
			format:    "docker",
		},
		{
			name:      "TC9 - normal case save multiple images with format docker",
			path:      tmpDir.Join("test9"),
			args:      []string{"testImage1", "testImage2"},
			wantErr:   true,
			errString: "isula_build.sock",
			format:    "docker",
		},
		{
			name:      "TC10 - abnormal case save multiple images with format oci",
			path:      tmpDir.Join("test10"),
			args:      []string{"testImage1", "testImage2"},
			wantErr:   true,
			errString: "oci image format now only supports saving single image",
			format:    "oci",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			saveCmd := NewSaveCmd()
			saveOpts = saveOptions{
				images: tc.args,
				path:   tc.path,
				format: tc.format,
			}
			err := saveCommand(saveCmd, saveOpts.images)
			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errString)
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
}

func TestRunSave(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	alreadyExistFile := fs.NewFile(t, tmpDir.Join("alreadyExist.tar"))
	defer alreadyExistFile.Remove()

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
			name: "TC2 - normal case with multiple image",
			path: tmpDir.Join("test2"),
			args: []string{"testImage1:test", "testImage2:test"},
		},
		{
			name: "TC3 - normal case with save failed",
			path: tmpDir.Join("test3"),
			args: []string{imageID, "testImage1:test"},
			// construct failed env when trying to save image id "38b993607bcabe01df1dffdf01b329005c6a10a36d557f9d073fc25943840c66"
			wantErr:   true,
			errString: "failed to save image 38b993607bcabe01df1dffdf01b329005c6a10a36d5",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockSave := newMockDaemon()
			cli := newMockClient(&mockGrpcClient{saveFunc: mockSave.save})
			saveOpts.path = tc.path
			err := runSave(ctx, &cli, tc.args)
			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errString)
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
}
