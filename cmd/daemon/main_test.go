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
// Create: 2020-01-20
// Description: This file is used for isula-build daemon testing

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/fs"

	constant "isula.org/isula-build"
)

func TestSetupWorkingDirectories(t *testing.T) {
	var testDir *fs.Dir
	var testcases = []struct {
		name        string
		prepareFunc func(t *testing.T)
		wantErr     bool
	}{
		{
			name: "TC1 - normal - new env",
			prepareFunc: func(t *testing.T) {
				testDir = fs.NewDir(t, "TestSetupWorkingDirectories")
				daemonOpts.DataRoot = testDir.Join("data")
				daemonOpts.RunRoot = testDir.Join("run")
			},
			wantErr: false,
		},
		{
			name: "TC2 - normal - already exist",
			prepareFunc: func(t *testing.T) {
				testDir = fs.NewDir(t, "TestSetupWorkingDirectories")
				daemonOpts.DataRoot = testDir.Join("data")
				daemonOpts.RunRoot = testDir.Join("run")
				os.Mkdir(daemonOpts.DataRoot, constant.DefaultSharedDirMode)
				os.Mkdir(daemonOpts.RunRoot, constant.DefaultSharedDirMode)
			},
			wantErr: false,
		},
		{
			name: "TC3 - abnormal - exist file with same name",
			prepareFunc: func(t *testing.T) {
				testDir = fs.NewDir(t, "TestSetupWorkingDirectories")
				daemonOpts.DataRoot = testDir.Join("data")
				daemonOpts.RunRoot = testDir.Join("run")
				os.Mkdir(daemonOpts.DataRoot, constant.DefaultSharedDirMode)
				ioutil.WriteFile(daemonOpts.RunRoot, []byte{}, constant.DefaultSharedFileMode)
			},
			wantErr: true,
		},
		{
			name: "TC4 - abnormal - exist file with same name 2",
			prepareFunc: func(t *testing.T) {
				testDir = fs.NewDir(t, "TestSetupWorkingDirectories")
				daemonOpts.DataRoot = testDir.Join("data")
				daemonOpts.RunRoot = testDir.Join("run")
				os.Mkdir(daemonOpts.RunRoot, constant.DefaultSharedDirMode)
				ioutil.WriteFile(daemonOpts.DataRoot, []byte{}, constant.DefaultSharedFileMode)
			},
			wantErr: true,
		},
		{
			name: "TC5 - abnormal - exist file with same name 3",
			prepareFunc: func(t *testing.T) {
				testDir = fs.NewDir(t, "TestSetupWorkingDirectories")
				daemonOpts.DataRoot = testDir.Join("data")
				daemonOpts.RunRoot = testDir.Join("run")
				ioutil.WriteFile(daemonOpts.DataRoot, []byte{}, constant.DefaultSharedFileMode)
				ioutil.WriteFile(daemonOpts.RunRoot, []byte{}, constant.DefaultSharedFileMode)
			},
			wantErr: true,
		},
		{
			name: "TC6 - abnormal - Relative path",
			prepareFunc: func(t *testing.T) {
				daemonOpts.DataRoot = "foo/bar"
				daemonOpts.RunRoot = "foo/bar"
			},
			wantErr: true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareFunc(t)
			defer testDir.Remove()

			daemonOpts.Group = "root"
			if err := setupWorkingDirectories(); (err != nil) != tt.wantErr {
				t.Errorf("testing failed! err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
