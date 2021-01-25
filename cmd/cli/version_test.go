// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2020-01-20
// Description: This file is used for testing command version

package main

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	"isula.org/isula-build/pkg/version"
)

func TestVersionCommand(t *testing.T) {
	versionCmd := NewVersionCmd()
	var args []string
	err := versionCommand(versionCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestVersionCommandParseFail(t *testing.T) {
	versionCmd := NewVersionCmd()
	var args []string
	version.BuildInfo = "abc"
	err := versionCommand(versionCmd, args)
	assert.ErrorContains(t, err, "invalid syntax")
}

func TestGetDaemonVersion(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	err := getDaemonVersion(ctx, &cli)
	assert.NilError(t, err)
}
