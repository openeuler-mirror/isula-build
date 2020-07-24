// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zhongkai Lei
// Create: 2020-03-20
// Description: image related functions tests

package image

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

func TestFindImageWhenImageNameIsEmpty(t *testing.T) {
	runDir := fs.NewDir(t, "run",
		fs.WithDir("overlay",
			fs.WithFile("overlay-true", "")))
	libDir := fs.NewDir(t, "lib")

	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: libDir.Path(),
		RunRoot:  runDir.Path(),
	})

	localStore, err := store.GetStore()
	assert.NilError(t, err)

	defer func() {
		unix.Unmount(filepath.Join(runDir.Path(), "overlay"), 0)
		unix.Unmount(filepath.Join(libDir.Path(), "overlay"), 0)
		runDir.Remove()
		libDir.Remove()
	}()

	src := ""
	srcReference, _, err := FindImage(localStore, src)
	assert.ErrorContains(t, err, "locating image")
	assert.Assert(t, cmp.Nil(srcReference))
}

func TestTryResolveNameWithDockerReference(t *testing.T) {
	type testcase struct {
		name        string
		expectTrans string
		errStr      string
	}
	var testcases = []testcase{
		{
			name:        "docker.io/library/busybox:latest",
			expectTrans: util.DefaultTransport,
			errStr:      "",
		}, {
			name:        "busybox:latest",
			expectTrans: "",
			errStr:      "",
		}, {
			name:        "Busybox:latest",
			expectTrans: "",
			errStr:      "repository name must be lowercase",
		},
	}

	for _, tc := range testcases {
		name := tc.name
		_, transport, err := tryResolveNameWithDockerReference(name)
		assert.Equal(t, transport, tc.expectTrans)
		if err != nil {
			assert.ErrorContains(t, err, tc.errStr)
		}
	}
}

func TestTryResolveNameInRegistries(t *testing.T) {
	filename := "registries.conf"
	dir := "/etc/containers"
	filePath := filepath.Join(dir, filename)

	registriesCfg := `
[registries.search]
registries = ['docker.io']

[registries.insecure]
registries = []

[registries.block]
registries = []
`
	var err error
	if _, err = os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dir, constant.DefaultRootDirMode); err == nil {
			defer os.RemoveAll(dir)
		}
	}
	if err != nil {
		t.Skip("skingping test, because of:", err)
	}

	if _, err = os.Stat(filePath); err != nil && os.IsNotExist(err) {
		err = ioutil.WriteFile(filePath, []byte(registriesCfg), constant.DefaultRootFileMode)
		assert.NilError(t, err)
		defer os.Remove(filePath)
	}

	name := "busybox:latest"
	candidates, transport := tryResolveNameInRegistries(name, nil)
	assert.Assert(t, cmp.Contains(candidates, "localhost/busybox:latest"))
	assert.Equal(t, transport, util.DefaultTransport)
}
