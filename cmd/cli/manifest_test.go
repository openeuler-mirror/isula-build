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
// Create: 2020-12-01
// Description: This file is used for testing manifest command.

package main

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestManifestCommand(t *testing.T) {
	manifestCmd := NewManifestCmd()
	var args []string
	err := manifestCreateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "please specify a name to manifest list")

	args = []string{"openeuler"}
	err = manifestCreateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")

	args = []string{}
	err = manifestAnnotateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "please specify the manifest list and the image name")

	args = []string{"openeuler"}
	err = manifestAnnotateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "please specify the manifest list and the image name")

	args = []string{"openeuler", "aaa", "bbb"}
	err = manifestAnnotateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "please specify the manifest list and the image name")

	args = []string{"openeuler", "openeuler_x86"}
	err = manifestAnnotateCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")

	args = []string{}
	err = manifestInspectCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "please specify the manifest list name")

	args = []string{"openeuler", "bbb"}
	err = manifestInspectCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "only one manifest list can be specified")

	args = []string{"openeuler"}
	err = manifestInspectCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")

	args = []string{"openeuler", "localhost:5000/openeuler"}
	err = manifestPushCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")

	args = []string{"openeuler"}
	err = manifestPushCommand(manifestCmd, args)
	assert.ErrorContains(t, err, "specify the manifest list name and destination repository")
}

func TestRunManifestCreate(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	listName := "openeuler"
	manifestsName := []string{"openeuler_x86"}
	err := runManifestCreate(ctx, &cli, listName, manifestsName)
	assert.NilError(t, err)
}

func TestRunManifestAnnotate(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	listName := "openeuler"
	manifestsName := "openeuler_x86"
	err := runManifestAnnotate(ctx, &cli, listName, manifestsName)
	assert.NilError(t, err)
}

func TestRunManifestInspect(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	listName := "openeuler"
	err := runManifestInspect(ctx, &cli, listName)
	assert.NilError(t, err)
}

func TestRunManifestPush(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	listName := "openeuler"
	dest := "localhost:5000/openeuler"
	err := runManifestPush(ctx, &cli, listName, dest)
	assert.NilError(t, err)
}
