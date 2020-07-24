// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-01-20
// Description: This is test file for daemon

package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"

	constant "isula.org/isula-build"
)

func TestSetDaemonLock(t *testing.T) {
	root := "/tmp/this_is_a_test_folder"
	name := "test.lock"
	lockPath := filepath.Join(root, name)

	// when folder is not exist, daemon lock is not supposed to be set
	_, err := setDaemonLock(root, name)
	assert.ErrorContains(t, err, "no such file or directory")

	// create lockfile
	err = os.Mkdir(root, constant.DefaultRootDirMode)
	defer os.RemoveAll(root)
	assert.NilError(t, err)
	f, err := os.Create(lockPath)
	assert.NilError(t, err)
	defer f.Close()
	// set daemon lock successful
	_, err = setDaemonLock(root, name)
	assert.NilError(t, err)

	// set daemon lock twice will fail
	_, err = setDaemonLock(root, name)
	assert.ErrorContains(t, err, "check if there is another daemon running")
}
