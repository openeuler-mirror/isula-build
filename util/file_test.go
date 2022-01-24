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
// Create: 2021-08-24
// Description: file manipulation related common functions

package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/storage/pkg/archive"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	constant "isula.org/isula-build"
)

func TestLoadJSONFile(t *testing.T) {
	type rename struct {
		Name   string `json:"name"`
		Rename string `json:"rename"`
	}
	type args struct {
		file string
		v    rename
	}

	smallJSONFile := fs.NewFile(t, t.Name())
	defer smallJSONFile.Remove()
	validData := rename{
		Name:   "origin name",
		Rename: "modified name",
	}
	b, err := json.Marshal(validData)
	assert.NilError(t, err)
	ioutil.WriteFile(smallJSONFile.Path(), b, constant.DefaultRootFileMode)

	tests := []struct {
		name      string
		args      args
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name: "TC-normal json file",
			args: args{
				file: smallJSONFile.Path(),
				v:    rename{},
			},
			wantKey:   "origin name",
			wantValue: "modified name",
		},
		{
			name:    "TC-json file not exist",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadJSONFile(tt.args.file, &tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("LoadJSONFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				assert.Equal(t, tt.args.v.Name, tt.wantKey)
				assert.Equal(t, tt.args.v.Rename, tt.wantValue)
			}
		})
	}
}

func TestChangeFileModifyTime(t *testing.T) {
	normalFile := fs.NewFile(t, t.Name())
	defer normalFile.Remove()

	pwd, err := os.Getwd()
	assert.NilError(t, err)
	immutableFile := filepath.Join(pwd, "immutableFile")
	_, err = os.Create(immutableFile)
	defer os.Remove(immutableFile)

	type args struct {
		path  string
		mtime time.Time
		atime time.Time
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		needHook    bool
		preHookFun  func(t *testing.T)
		postHookFun func(t *testing.T)
	}{
		{
			name: "TC-change file modify time",
			args: args{
				path:  immutableFile,
				mtime: modifyTime,
				atime: accessTime,
			},
		},
		{
			name:    "TC-file path empty",
			wantErr: true,
		},
		{
			name: "TC-lack of permession",
			args: args{
				path:  immutableFile,
				atime: accessTime,
				mtime: modifyTime,
			},
			needHook:    true,
			preHookFun:  func(t *testing.T) { Immutable(immutableFile, true) },
			postHookFun: func(t *testing.T) { Immutable(immutableFile, false) },
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.needHook {
				tt.preHookFun(t)
			}
			err := ChangeFileModifyTime(tt.args.path, tt.args.atime, tt.args.mtime)
			if tt.needHook {
				defer tt.postHookFun(t)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeFileModifyTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				f, err := os.Stat(tt.args.path)
				assert.NilError(t, err)
				assert.Equal(t, true, f.ModTime().Equal(modifyTime))
			}
		})
	}
}

type tempDirs struct {
	root     string
	subDir1  string
	subDir11 string
	file1    string
	file11   string
}

func createDirs(t *testing.T) tempDirs {
	pwd, err := os.Getwd()
	assert.NilError(t, err)

	root := filepath.Join(pwd, t.Name())
	assert.NilError(t, os.Mkdir(root, constant.DefaultRootDirMode))

	rootSubDir1 := filepath.Join(root, "rootSubDir1")
	assert.NilError(t, os.Mkdir(rootSubDir1, constant.DefaultRootDirMode))

	rootSubDir11 := filepath.Join(rootSubDir1, "rootSubDir11")
	assert.NilError(t, os.Mkdir(rootSubDir11, constant.DefaultRootDirMode))

	file1 := filepath.Join(rootSubDir1, "file1")
	_, err = os.Create(file1)
	assert.NilError(t, err)

	file11 := filepath.Join(rootSubDir11, "file11")
	_, err = os.Create(file11)
	assert.NilError(t, err)

	return tempDirs{
		root:     root,
		subDir1:  rootSubDir1,
		subDir11: rootSubDir11,
		file1:    file1,
		file11:   file11,
	}
}

func (tmp *tempDirs) removeAll(t *testing.T) {
	assert.NilError(t, os.RemoveAll(tmp.root))
	assert.NilError(t, os.RemoveAll(tmp.subDir1))
	assert.NilError(t, os.RemoveAll(tmp.subDir11))
	assert.NilError(t, os.RemoveAll(tmp.file1))
	assert.NilError(t, os.RemoveAll(tmp.file11))
}

func TestChangeDirModifyTime(t *testing.T) {
	tempDirs := createDirs(t)
	defer tempDirs.removeAll(t)
	root := tempDirs.root

	type args struct {
		dir   string
		mtime time.Time
		atime time.Time
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		needPreHook  bool
		needPostHook bool
		preWalkFun   func(path string, info os.FileInfo, err error) error
		postWalkFun  func(path string, info os.FileInfo, err error) error
	}{
		{
			name: "TC-normal case modify directory",
			args: args{
				dir:   root,
				mtime: modifyTime,
				atime: accessTime,
			},
			needPostHook: true,
			postWalkFun: func(path string, info os.FileInfo, err error) error {
				assert.Assert(t, true, info.ModTime().Equal(modifyTime))
				return nil
			},
		},
		{
			name:    "TC-empty path",
			wantErr: true,
		},
		{
			name: "TC-lack of permission",
			args: args{
				dir:   root,
				mtime: modifyTime,
				atime: accessTime,
			},
			wantErr:      true,
			needPreHook:  true,
			needPostHook: true,
			preWalkFun: func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					Immutable(path, true)
				}
				return nil
			},
			postWalkFun: func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					Immutable(path, false)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.needPreHook {
				wErr := filepath.Walk(tt.args.dir, tt.preWalkFun)
				assert.NilError(t, wErr)
			}
			err := ChangeDirModifyTime(tt.args.dir, tt.args.mtime, tt.args.atime)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeDirModifyTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.needPostHook {
				wErr := filepath.Walk(tt.args.dir, tt.postWalkFun)
				assert.NilError(t, wErr)
			}
		})
	}
}

func TestPackFiles(t *testing.T) {
	dirs := createDirs(t)
	defer dirs.removeAll(t)
	dest := fs.NewFile(t, t.Name())
	defer dest.Remove()

	type args struct {
		src            string
		dest           string
		com            archive.Compression
		needModifyTime bool
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		needPreHook  bool
		needPostHook bool
		preWalkFun   func(path string, info os.FileInfo, err error) error
		postWalkFun  func(path string, info os.FileInfo, err error) error
	}{
		{
			name: "TC-normal pack",
			args: args{
				src:            dirs.root,
				dest:           dest.Path(),
				com:            archive.Gzip,
				needModifyTime: true,
			},
		},
		{
			name: "TC-empty dest",
			args: args{
				src:            dirs.root,
				com:            archive.Gzip,
				needModifyTime: true,
			},
			wantErr: true,
		},
		{
			name: "TC-invalid compression",
			args: args{
				src:            dirs.root,
				dest:           dest.Path(),
				com:            archive.Compression(-1),
				needModifyTime: true,
			},
			wantErr: true,
		},
		{
			name: "TC-lack of permission",
			args: args{
				src:            dirs.root,
				dest:           dest.Path(),
				com:            archive.Gzip,
				needModifyTime: true,
			},
			wantErr:      true,
			needPreHook:  true,
			needPostHook: true,
			preWalkFun: func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					Immutable(path, true)
				}
				return nil
			},
			postWalkFun: func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					Immutable(path, false)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.needPreHook {
				wErr := filepath.Walk(tt.args.src, tt.preWalkFun)
				assert.NilError(t, wErr)
			}
			if err := PackFiles(tt.args.src, tt.args.dest, tt.args.com, tt.args.needModifyTime); (err != nil) != tt.wantErr {
				t.Errorf("PackFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.needPostHook {
				wErr := filepath.Walk(tt.args.src, tt.postWalkFun)
				assert.NilError(t, wErr)
			}
		})
	}
}

func TestUnpackFile(t *testing.T) {
	folderToBePacked := createDirs(t)
	defer folderToBePacked.removeAll(t)
	pwd, err := os.Getwd()
	assert.NilError(t, err)

	tarName := filepath.Join(pwd, "test.tar")
	assert.NilError(t, PackFiles(folderToBePacked.root, tarName, archive.Gzip, true))
	defer os.RemoveAll(tarName)

	invalidTar := filepath.Join(pwd, "invalid.tar")
	err = ioutil.WriteFile(invalidTar, []byte("invalid tar"), constant.DefaultRootFileMode)
	assert.NilError(t, err)
	defer os.RemoveAll(invalidTar)

	unpackDest := filepath.Join(pwd, "unpack")
	assert.NilError(t, os.MkdirAll(unpackDest, constant.DefaultRootDirMode))
	defer os.RemoveAll(unpackDest)

	type args struct {
		src  string
		dest string
		com  archive.Compression
		rm   bool
	}
	tests := []struct {
		name         string
		args         args
		needPreHook  bool
		needPostHook bool
		wantErr      bool
	}{
		{
			name: "normal unpack file",
			args: args{
				src:  tarName,
				dest: unpackDest,
				com:  archive.Gzip,
				rm:   true,
			},
		},
		{
			name: "empty unpack destation path",
			args: args{
				src: tarName,
				com: archive.Gzip,
				rm:  false,
			},
			wantErr: true,
		},
		{
			name: "unpack src path not exist",
			args: args{
				src:  "path not exist",
				dest: unpackDest,
				com:  archive.Gzip,
				rm:   false,
			},
			wantErr: true,
		},
		{
			name: "unpack destation path not exist",
			args: args{
				src:  tarName,
				dest: "path not exist",
				com:  archive.Gzip,
				rm:   false,
			},
			wantErr: true,
		},
		{
			name: "invalid tarball",
			args: args{
				src:  invalidTar,
				dest: unpackDest,
				com:  archive.Gzip,
				rm:   false,
			},
			wantErr: true,
		},
		{
			name: "no permission for src",
			args: args{
				src:  tarName,
				dest: unpackDest,
				com:  archive.Gzip,
				rm:   true,
			},
			wantErr:      true,
			needPreHook:  true,
			needPostHook: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.needPreHook {
				assert.NilError(t, Immutable(tt.args.src, true))
			}
			err := UnpackFile(tt.args.src, tt.args.dest, tt.args.com, tt.args.rm)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.needPostHook {
				assert.NilError(t, Immutable(tt.args.src, false))
			}
			if tt.args.rm && err == nil {
				tarName := filepath.Join(pwd, "test.tar")
				assert.NilError(t, PackFiles(folderToBePacked.root, tarName, archive.Gzip, true))
			}
		})
	}
}
