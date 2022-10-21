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
// Create: 2022-10-20
// Description: This file is for image import test.

// Description: This file is is used for testing import command.
package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
)

const ExceededImportFileSize = 2048 * 1024 * 1024

func TestImportCommand(t *testing.T) {
	tmpFile := fs.NewFile(t, t.Name())
	exceededFile := fs.NewFile(t, t.Name())
	err := ioutil.WriteFile(tmpFile.Path(), []byte("This is test file"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	err = ioutil.WriteFile(exceededFile.Path(), []byte("This is exceeded test file"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	err = os.Truncate(exceededFile.Path(), ExceededImportFileSize)
	assert.NilError(t, err)
	defer tmpFile.Remove()
	defer exceededFile.Remove()

	type testcase struct {
		name      string
		errString string
		args      []string
		wantErr   bool
	}
	var testcases = []testcase{
		{
			name:      "TC1 - abnormal case with no args",
			errString: "requires at least one argument",
			wantErr:   true,
		},
		{
			name:      "TC2 - abnormal case with exceeded limit input",
			args:      []string{exceededFile.Path()},
			errString: "exceeds limit 1073741824",
			wantErr:   true,
		},
		{
			name:      "TC3 - normal case",
			args:      []string{tmpFile.Path()},
			errString: "isula_build.sock",
			wantErr:   true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			importCmd := NewImportCmd()
			err = importCommand(importCmd, tc.args)
			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errString)
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
}

func TestRunImport(t *testing.T) {
	ctx := context.Background()
	mockImport := newMockDaemon()
	cli := newMockClient(&mockGrpcClient{importFunc: mockImport.importImage})
	fileEmpty := "empty.tar"
	fileNormal := "test.tar"
	exceededFile := fs.NewFile(t, t.Name())
	err := ioutil.WriteFile(exceededFile.Path(), []byte("This is exceeded test file"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	err = os.Truncate(exceededFile.Path(), ExceededImportFileSize)
	assert.NilError(t, err)
	ctxDir := fs.NewDir(t, "import", fs.WithFile(fileEmpty, ""), fs.WithFile(fileNormal, "test"))
	defer ctxDir.Remove()
	defer exceededFile.Remove()

	type testcase struct {
		name      string
		source    string
		wantErr   bool
		errString string
	}
	var testcases = []testcase{
		{
			name:      "TC1 - abnormal case with empty file",
			source:    filepath.Join(ctxDir.Path(), fileEmpty),
			wantErr:   true,
			errString: "empty",
		},
		{
			name:      "TC2 - abnormal case with exceeded limit file",
			source:    exceededFile.Path(),
			wantErr:   true,
			errString: "limit",
		},
		{
			name:    "TC3 - normal case",
			source:  filepath.Join(ctxDir.Path(), fileNormal),
			wantErr: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			importOpts.source = tc.source
			err := runImport(ctx, &cli)
			assert.Equal(t, err != nil, tc.wantErr, "Failed at [%s], err: %v", tc.name, err)
			if err != nil {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}
