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
// Create: 2020-01-20
// Description: store related functions tests

package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
)

func init() {
	SetDefaultStoreOptions(DaemonStoreOptions{
		DataRoot: "/tmp/isula-build/store/data",
		RunRoot:  "/tmp/isula-build/store/run",
	})
	reexec.Init()
}

func TestGetDefaultStoreOptions(t *testing.T) {
	_, err := GetDefaultStoreOptions()
	assert.NilError(t, err)
}

func TestGetStorageConfigFileOptions(t *testing.T) {
	_, err := GetStorageConfigFileOptions()
	assert.NilError(t, err)
}

func TestGetStore(t *testing.T) {
	dataDir := "/tmp/lib"
	runDir := "/tmp/run"
	storeOpts.DataRoot = filepath.Join(dataDir, "containers/storage")
	storeOpts.RunRoot = filepath.Join(runDir, "containers/storage")

	s, err := GetStore()
	assert.NilError(t, err)
	defer func() {
		unix.Unmount(filepath.Join(storeOpts.DataRoot, "overlay"), 0)
		unix.Unmount(filepath.Join(storeOpts.RunRoot, "overlay"), 0)
		os.RemoveAll(dataDir)
		os.RemoveAll(runDir)
	}()
	assert.Equal(t, s.RunRoot(), storeOpts.RunRoot)
	assert.Equal(t, s.GraphRoot(), storeOpts.DataRoot)
}

func TestCleanContainers(t *testing.T) {
	dataDir := "/tmp/lib"
	runDir := "/tmp/run"
	storeOpts.DataRoot = filepath.Join(dataDir, "containers/storage")
	storeOpts.RunRoot = filepath.Join(runDir, "containers/storage")

	s, err := GetStore()
	assert.NilError(t, err)
	s.CreateContainer("", []string{""}, "", "TC1", "", nil)
	s.CleanContainers()
	containers, _ := s.Containers()
	if len(containers) > 0 {
		t.Errorf("Failed to clean containers")
	}
	s.CreateContainer("", []string{""}, "", "TC2", "", nil)
	err = os.RemoveAll(filepath.Join(storeOpts.DataRoot, "overlay-layers"))
	assert.NilError(t, err)
	s.CleanContainers()

	defer func() {
		unix.Unmount(filepath.Join(storeOpts.DataRoot, "overlay"), 0)
		unix.Unmount(filepath.Join(storeOpts.RunRoot, "overlay"), 0)
		os.RemoveAll(dataDir)
		os.RemoveAll(runDir)
	}()
}

func TestCleanContainer(t *testing.T) {
	dataDir := "/tmp/lib"
	runDir := "/tmp/run"
	storeOpts.DataRoot = filepath.Join(dataDir, "containers/storage")
	storeOpts.RunRoot = filepath.Join(runDir, "containers/storage")

	s, err := GetStore()
	assert.NilError(t, err)
	s.CreateContainer("", []string{""}, "", "TC1", "", nil)
	err = s.CleanContainer("TC1")
	assert.NilError(t, err)
	err = s.CleanContainer("TC2")
	assert.ErrorContains(t, err, "not a container")

	defer func() {
		unix.Unmount(filepath.Join(storeOpts.DataRoot, "overlay"), 0)
		unix.Unmount(filepath.Join(storeOpts.RunRoot, "overlay"), 0)
		os.RemoveAll(dataDir)
		os.RemoveAll(runDir)
	}()
}
