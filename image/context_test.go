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
// Description: image context related functions tests

package image

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	constant "isula.org/isula-build"
)

func doCmd(cmd string) {
	if cmd != "" {
		cmd := exec.Command("/bin/sh", "-c", cmd)
		cmd.Run()
	}
}

func TestSetSystemContext(t *testing.T) {
	prepareFunc := func(path string) {
		if _, err := os.Stat(path); err != nil {
			doCmd(fmt.Sprintf("echo hello > %s", path))
			defer func() {
				doCmd(fmt.Sprintf("rm -f %s", path))
			}()
		}
	}

	prepareFunc(constant.SignaturePolicyPath)
	prepareFunc(constant.RegistryConfigPath)
	dataRoot := constant.DefaultDataRoot

	SetSystemContext(dataRoot)
}

func TestGetSystemContext(t *testing.T) {
	// make sure the return value is a pointer and comparable
	// at the same time, test for COPY action
	sc1 := GetSystemContext()
	sc2 := GetSystemContext()
	if sc1 == sc2 {
		t.Errorf("test failed for GetSystemContext")
	}
}
