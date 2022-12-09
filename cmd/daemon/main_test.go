// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2022-12-08
// Description: This file is used for isula-build cmd/daemon testing

package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestRunDaemon(t *testing.T) {
	cmd := newDaemonCommand()
	daemonOpts.Group = "none"
	err := runDaemon(cmd, []string{})
	assert.ErrorContains(t, err, "create new GRPC socket failed")
}
