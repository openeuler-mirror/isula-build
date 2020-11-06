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
// Create: 2020-11-02
// Description: This file is used for testing command pull

package main

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestPullCommand(t *testing.T) {
	pullCmd := NewPullCmd()
	args := []string{"openeuler:latest"}
	err := pullCommand(pullCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestPullCommandMultipleArgs(t *testing.T) {
	pullCmd := NewPullCmd()
	args := []string{"aaa", "bbb"}
	err := pullCommand(pullCmd, args)
	assert.ErrorContains(t, err, "pull requires exactly one argument")
}

func TestRunPull(t *testing.T) {
	ctx := context.Background()
	mockPull := newMockDaemon()
	cli := newMockClient(&mockGrpcClient{pullFunc: mockPull.pull})
	err := runPull(ctx, &cli, "")
	assert.NilError(t, err)
}
