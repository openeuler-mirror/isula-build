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
// Description: This file is used for testing command image

package main

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestImageCommand(t *testing.T) {
	imageCmd := NewImagesCmd()
	var args []string
	err := imagesCommand(imageCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestImageCommandMultipleArgs(t *testing.T) {
	imageCmd := NewImagesCmd()
	args := []string{"aaa", "bbb"}
	err := imagesCommand(imageCmd, args)
	assert.ErrorContains(t, err, "requires at most one argument")
}

func TestRunList(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	err := runList(ctx, &cli, "")
	assert.NilError(t, err)
}
