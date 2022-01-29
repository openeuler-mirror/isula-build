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
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
	"isula.org/isula-build/cmd/daemon/config"
	"isula.org/isula-build/store"
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

func TestRunAndDataRootSet(t *testing.T) {
	dataRoot := fs.NewDir(t, t.Name())
	runRoot := fs.NewDir(t, t.Name())
	result := store.DaemonStoreOptions{
		DataRoot: dataRoot.Join("storage"),
		RunRoot:  runRoot.Join("storage"),
	}

	conf := config.TomlConfig{
		Debug:    true,
		Group:    "isula",
		LogLevel: "debug",
		Runtime:  "",
		RunRoot:  "",
		DataRoot: "",
	}
	cmd := newDaemonCommand()

	setStorage := func(content string) {
		// merge main config
		if err := mergeConfig(conf, cmd); err != nil {
			t.Fatalf("merge config failed with error: %v", err)
		}

		// simulate storage.toml and merge
		fileName := "storage.toml"
		tmpDir := fs.NewDir(t, t.Name(), fs.WithFile(fileName, content))
		defer tmpDir.Remove()
		filePath := tmpDir.Join(fileName)

		store.SetDefaultConfigFilePath(filePath)
		option, err := store.GetDefaultStoreOptions(true)
		if err != nil {
			t.Fatalf("get default store options failed with error: %v", err)
		}

		if !cmd.Flag("runroot").Changed && option.RunRoot != "" {
			daemonOpts.RunRoot = option.RunRoot
		}
		if !cmd.Flag("dataroot").Changed && option.GraphRoot != "" {
			daemonOpts.DataRoot = option.GraphRoot
		}
		if !cmd.Flag("storage-driver").Changed && option.GraphDriverName != "" {
			daemonOpts.StorageDriver = option.GraphDriverName
		}
		if !cmd.Flag("storage-opt").Changed && len(option.GraphDriverOptions) > 0 {
			daemonOpts.StorageOpts = option.GraphDriverOptions
		}

		// final set
		if err := setStoreAccordingToDaemonOpts(); err != nil {
			t.Fatalf("set store options failed with error: %v", err)
		}
	}

	testcases := []struct {
		name        string
		setF        func()
		expectation store.DaemonStoreOptions
	}{
		{
			// first run so can not be affected by other testcase
			name: "TC1 - all not set",
			setF: func() { setStorage("[storage]\ndriver = \"overlay\"") },
			expectation: store.DaemonStoreOptions{
				DataRoot: "/var/lib/isula-build/storage",
				RunRoot:  "/var/run/isula-build/storage",
			},
		},
		{
			name: "TC2 - cmd set, configuration and storage not set",
			setF: func() {
				cmd.PersistentFlags().Set("runroot", runRoot.Path())
				cmd.PersistentFlags().Set("dataroot", dataRoot.Path())
				setStorage("[storage]\ndriver = \"overlay\"")
			},
			expectation: result,
		},
		{
			name: "TC3 - cmd and storage not set, configuration set",
			setF: func() {
				conf.DataRoot = dataRoot.Path()
				conf.RunRoot = runRoot.Path()
				setStorage("[storage]\ndriver = \"overlay\"")
			},
			expectation: result,
		},
		{
			name: "TC4 - cmd and configuration not set, storage set",
			setF: func() {
				config := fmt.Sprintf("[storage]\ndriver = \"%s\"\nrunroot = \"%s\"\ngraphroot = \"%s\"\n",
					"overlay", runRoot.Join("storage"), dataRoot.Join("storage"))
				setStorage(config)
			},
			expectation: result,
		},
		{
			name: "TC5 - cmd not set, configuration and storage set, configuration first",
			setF: func() {
				conf.DataRoot = dataRoot.Path()
				conf.RunRoot = runRoot.Path()

				dataRootStorage := fs.NewDir(t, t.Name())
				runRootStorage := fs.NewDir(t, t.Name())
				config := fmt.Sprintf("[storage]\ndriver = \"%s\"\nrunroot = \"%s\"\ngraphroot = \"%s\"\n",
					"overlay", runRootStorage.Join("storage"), dataRootStorage.Join("storage"))

				setStorage(config)
			},
			expectation: result,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setF()
			storeOptions, err := store.GetDefaultStoreOptions(false)
			if err != nil {
				t.Fatalf("get default store options failed with error: %v", err)
			}
			assert.Equal(t, tc.expectation.DataRoot, storeOptions.GraphRoot)
			assert.Equal(t, tc.expectation.RunRoot, storeOptions.RunRoot)
		})

	}
}

func TestValidateConfigFileAndMerge(t *testing.T) {
	// cmd line args keep default.
	cmd := newDaemonCommand()
	err := validateConfigFileAndMerge(cmd)
	assert.NilError(t, err)

	// cmd line runroot and dataroot args are set.
	dataRoot := fs.NewDir(t, t.Name())
	runRoot := fs.NewDir(t, t.Name())
	cmd.PersistentFlags().Set("runroot", runRoot.Path())
	cmd.PersistentFlags().Set("dataroot", dataRoot.Path())
	err = validateConfigFileAndMerge(cmd)
	assert.NilError(t, err)

	if err := setStoreAccordingToDaemonOpts(); err != nil {
		t.Fatalf("set store options failed with error: %v", err)
	}
	storeOptions, err := store.GetDefaultStoreOptions(false)
	if err != nil {
		t.Fatalf("get default store options failed with error: %v", err)
	}

	assert.Equal(t, storeOptions.GraphRoot, dataRoot.Join("storage"))
	assert.Equal(t, storeOptions.RunRoot, runRoot.Join("storage"))
}
