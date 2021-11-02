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
// Create: 2021-11-02
// Description: testcase for filepath related common functions

package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	constant "isula.org/isula-build"
)

func TestIsExist(t *testing.T) {
	type args struct {
		path       string
		workingDir string
	}
	tests := []struct {
		name     string
		args     args
		want     string
		isExist  bool
		wantErr  bool
		preHook  func(t *testing.T, path string)
		postHook func(t *testing.T)
	}{
		{
			name: "TC-filename too long",
			args: args{
				path:       strings.Repeat("a", 256),
				workingDir: "/tmp",
			},
			want:    filepath.Join("/tmp", strings.Repeat("a", 256)),
			isExist: false,
			wantErr: true,
		},
		{
			name: "TC-filename valid",
			args: args{
				path:       strings.Repeat("a", 255),
				workingDir: "/tmp",
			},
			want:    filepath.Join("/tmp", strings.Repeat("a", 255)),
			isExist: false,
			wantErr: false,
		},
		{
			name: "TC-path too long",
			args: args{
				path:       strings.Repeat(strings.Repeat("a", 256)+"/", 16),
				workingDir: "/tmp",
			},
			want:    filepath.Join("/tmp", strings.Repeat(strings.Repeat("a", 256)+"/", 16)) + "/",
			isExist: false,
			wantErr: true,
		},
		{
			name: "TC-path exist",
			args: args{
				path:       strings.Repeat(strings.Repeat("a", 255)+"/", 15),
				workingDir: "/tmp",
			},
			want:    filepath.Join("/tmp", strings.Repeat(strings.Repeat("a", 255)+"/", 15)) + "/",
			isExist: true,
			wantErr: false,
			preHook: func(t *testing.T, path string) {
				err := os.MkdirAll(path, constant.DefaultRootDirMode)
				assert.NilError(t, err)
			},
			postHook: func(t *testing.T) {
				err := os.RemoveAll(filepath.Join("/tmp", strings.Repeat("a", 255)+"/"))
				assert.NilError(t, err)
			},
		},
		{
			name: "TC-path with dot exist",
			args: args{
				path:       ".",
				workingDir: filepath.Join("/tmp", strings.Repeat("./"+strings.Repeat("a", 255)+"/", 15)),
			},
			want:    filepath.Join("/tmp", strings.Repeat(strings.Repeat("a", 255)+"/", 15)) + "/",
			isExist: true,
			wantErr: false,
			preHook: func(t *testing.T, path string) {
				err := os.MkdirAll(path, constant.DefaultRootDirMode)
				assert.NilError(t, err)
			},
			postHook: func(t *testing.T) {
				err := os.RemoveAll(filepath.Join("/tmp", strings.Repeat("a", 255)+"/"))
				assert.NilError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeAbsolute(tt.args.path, tt.args.workingDir)
			if got != tt.want {
				t.Errorf("MakeAbsolute() = %v, want %v", got, tt.want)
				t.Skip()
			}

			if tt.preHook != nil {
				tt.preHook(t, got)
			}
			exist, err := IsExist(got)
			if exist != tt.isExist {
				t.Errorf("IsExist() = %v, want %v", exist, tt.isExist)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("IsExist() = %v, want %v", err, tt.wantErr)
			}
			if tt.postHook != nil {
				tt.postHook(t)
			}
		})
	}
}

func TestIsSymbolFile(t *testing.T) {
	originFile := "/tmp/originFile"
	symbolFile := "/tmp/symbolFile"
	noneExistFile := "/tmp/none_exist_file"
	type args struct {
		path string
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		preHook  func(t *testing.T)
		postHook func(t *testing.T)
	}{
		{
			name: "TC-is symbol file",
			args: args{path: "/tmp/symbolFile"},
			want: true,
			preHook: func(t *testing.T) {
				_, err := os.Create(originFile)
				assert.NilError(t, err)
				assert.NilError(t, os.Symlink(originFile, symbolFile))
			},
			postHook: func(t *testing.T) {
				assert.NilError(t, os.RemoveAll(originFile))
				assert.NilError(t, os.RemoveAll(symbolFile))
			},
		},
		{
			name: "TC-is normal file",
			args: args{path: originFile},
			want: false,
			preHook: func(t *testing.T) {
				_, err := os.Create(originFile)
				assert.NilError(t, err)
			},
			postHook: func(t *testing.T) {
				assert.NilError(t, os.RemoveAll(originFile))
			},
		},
		{
			name: "TC-file not exist",
			args: args{path: noneExistFile},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preHook != nil {
				tt.preHook(t)
			}
			if got := IsSymbolFile(tt.args.path); got != tt.want {
				t.Errorf("IsSymbolFile() = %v, want %v", got, tt.want)
			}
			if tt.postHook != nil {
				tt.postHook(t)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	dirPath := filepath.Join("/tmp", t.Name())
	filePath := filepath.Join("/tmp", t.Name())
	noneExistFile := "/tmp/none_exist_file"

	type args struct {
		path string
	}
	tests := []struct {
		name     string
		args     args
		want     bool
		preHook  func(t *testing.T)
		postHook func(t *testing.T)
	}{
		{
			name: "TC-is directory",
			args: args{path: dirPath},
			preHook: func(t *testing.T) {
				assert.NilError(t, os.MkdirAll(dirPath, constant.DefaultRootDirMode))
			},
			postHook: func(t *testing.T) {
				assert.NilError(t, os.RemoveAll(dirPath))
			},
			want: true,
		},
		{
			name: "TC-is file",
			args: args{path: dirPath},
			preHook: func(t *testing.T) {
				_, err := os.Create(filePath)
				assert.NilError(t, err)
			},
			postHook: func(t *testing.T) {
				assert.NilError(t, os.RemoveAll(filePath))
			},
		},
		{
			name: "TC-path not exist",
			args: args{path: noneExistFile},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preHook != nil {
				tt.preHook(t)
			}
			if got := IsDirectory(tt.args.path); got != tt.want {
				t.Errorf("IsDirectory() = %v, want %v", got, tt.want)
			}
			if tt.postHook != nil {
				tt.postHook(t)
			}
		})
	}
}
