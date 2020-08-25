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
// Description: This file is used for testing command remove

package main

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestRemoveCommand(t *testing.T) {
	removeCmd := NewRemoveCmd()
	var args []string
	err := removeCommand(removeCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestRunRemove(t *testing.T) {
	type testcase struct {
		name      string
		args      []string
		all       bool
		prune     bool
		errString string
		isErr     bool
	}
	var testcases = []testcase{
		{
			name:      "test 1",
			all:       false,
			prune:     false,
			errString: "imageID/name must be specified",
			isErr:     true,
		},
		{
			name:  "test 2",
			all:   false,
			prune: true,
			isErr: false,
		},
		{
			name:  "test 3",
			all:   true,
			prune: false,
			isErr: false,
		},
		{
			name:      "test 4",
			all:       true,
			prune:     true,
			errString: "--prune is not allowed when using --all",
			isErr:     true,
		},
		{
			name:  "test 5",
			args:  []string{"abc"},
			all:   false,
			prune: false,
			isErr: false,
		},
		{
			name:      "test 6",
			args:      []string{"abc"},
			all:       false,
			prune:     true,
			errString: "imageID/name is not allowed when using --prune",
			isErr:     true,
		},
		{
			name:      "test 7",
			args:      []string{"abc"},
			all:       true,
			prune:     false,
			errString: "imageID/name is not allowed when using --all",
			isErr:     true,
		},
		{
			name:      "test 8",
			args:      []string{"abc"},
			all:       true,
			prune:     true,
			errString: "imageID/name is not allowed when using --all",
			isErr:     true,
		},
	}
	for _, tc := range testcases {
		ctx := context.Background()
		mockRemove := newMockDaemon()
		cli := newMockClient(&mockGrpcClient{removeFunc: mockRemove.remove})
		removeOpts.all = tc.all
		removeOpts.prune = tc.prune
		err := runRemove(ctx, &cli, tc.args)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err != nil {
			assert.ErrorContains(t, err, tc.errString)
		}
	}
}
