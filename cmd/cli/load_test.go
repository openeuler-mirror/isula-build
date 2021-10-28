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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	constant "isula.org/isula-build"
)

func TestLoadCmd(t *testing.T) {
	tmpDir := fs.NewFile(t, t.Name())
	err := ioutil.WriteFile(tmpDir.Path(), []byte("This is test file"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	defer tmpDir.Remove()

	type testcase struct {
		name      string
		path      string
		errString string
		args      []string
		wantErr   bool
		sep       separatorLoadOption
	}
	// For normal cases, default err is "invalid socket path: unix:///var/run/isula_build.sock".
	// As daemon is not running as we run unit test.
	var testcases = []testcase{
		{
			name:      "TC1 - normal case",
			path:      tmpDir.Path(),
			errString: "isula_build.sock",
			wantErr:   true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			loadCmd := NewLoadCmd()
			loadOpts = loadOptions{
				path: tc.path,
				sep:  tc.sep,
			}
			err := loadCmd.Execute()
			assert.Equal(t, err != nil, true)

			err = loadCommand(loadCmd, tc.args)
			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errString)
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
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

func TestResolveLoadPath(t *testing.T) {
	dir := fs.NewDir(t, t.Name())
	fileWithContent := fs.NewFile(t, filepath.Join(t.Name(), "test.tar"))
	ioutil.WriteFile(fileWithContent.Path(), []byte("This is test file"), constant.DefaultRootFileMode)
	emptyFile := fs.NewFile(t, filepath.Join(t.Name(), "empty.tar"))

	defer dir.Remove()
	defer fileWithContent.Remove()
	defer emptyFile.Remove()

	type args struct {
		path string
		pwd  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TC-normal load path",
			args: args{
				path: fileWithContent.Path(),
				pwd:  dir.Path(),
			},
			want: fileWithContent.Path(),
		},
		{
			name: "TC-empty load path",
			args: args{
				pwd: dir.Path(),
			},
			wantErr: true,
		},
		{
			name: "TC-empty load file",
			args: args{
				path: emptyFile.Path(),
				pwd:  dir.Path(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveLoadPath(tt.args.path, tt.args.pwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveLoadPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveLoadPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckLoadOpts(t *testing.T) {
	root := fs.NewDir(t, t.Name())
	defer root.Remove()
	emptyFile, err := os.Create(filepath.Join(root.Path(), "empty.tar"))
	assert.NilError(t, err)
	fileWithContent, err := os.Create(filepath.Join(root.Path(), "test.tar"))
	assert.NilError(t, err)
	ioutil.WriteFile(fileWithContent.Name(), []byte("This is test file"), constant.DefaultRootFileMode)
	baseFile, err := os.Create(filepath.Join(root.Path(), "base.tar"))
	assert.NilError(t, err)
	ioutil.WriteFile(baseFile.Name(), []byte("This is base file"), constant.DefaultRootFileMode)
	libFile, err := os.Create(filepath.Join(root.Path(), "lib.tar"))
	ioutil.WriteFile(libFile.Name(), []byte("This is lib file"), constant.DefaultRootFileMode)

	type fields struct {
		path   string
		loadID string
		sep    separatorLoadOption
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "TC-normal load options",
			fields: fields{
				path: fileWithContent.Name(),
			},
		},
		{
			name:    "TC-empty load path",
			wantErr: true,
		},
		{
			name: "TC-empty load file",
			fields: fields{
				path: emptyFile.Name(),
			},
			wantErr: true,
		},
		{
			name: "TC-separated load",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					dir:  root.Path(),
					app:  "app:latest",
					base: baseFile.Name(),
					lib:  libFile.Name(),
				},
			},
		},
		{
			name: "TC-separated load with empty app name",
			fields: fields{
				sep: separatorLoadOption{
					dir:  root.Path(),
					base: baseFile.Name(),
					lib:  libFile.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with empty dir",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					base: baseFile.Name(),
					lib:  libFile.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with invalid app name",
			fields: fields{
				path: "invalid:app:name",
				sep: separatorLoadOption{
					dir:  root.Path(),
					base: baseFile.Name(),
					lib:  libFile.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with empty base tarball",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					dir:  root.Path(),
					base: emptyFile.Name(),
					lib:  libFile.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with empty lib tarball",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					dir:  root.Path(),
					base: baseFile.Name(),
					lib:  emptyFile.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with same base and lib tarball",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					dir:  root.Path(),
					base: fileWithContent.Name(),
					lib:  fileWithContent.Name(),
				},
			},
			wantErr: true,
		},
		{
			name: "TC-separated load with dir not exist",
			fields: fields{
				path: "app:latest",
				sep: separatorLoadOption{
					dir:  "path not exist",
					base: baseFile.Name(),
					lib:  libFile.Name(),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &loadOptions{
				path:   tt.fields.path,
				loadID: tt.fields.loadID,
				sep:    tt.fields.sep,
			}
			if err := opt.checkLoadOpts(); (err != nil) != tt.wantErr {
				t.Errorf("loadOptions.checkLoadOpts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
