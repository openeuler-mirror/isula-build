// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2020-07-20
// Description: This file is used for testing command tag

package main

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestTagCommand(t *testing.T) {
	tagCmd := NewTagCmd()
	args := []string{"abc", "abc"}
	err := tagCommand(tagCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")

	args = []string{"abc", "abc", "abc"}
	err = tagCommand(tagCmd, args)
	assert.ErrorContains(t, err, "invalid args for tag command")

	args = []string{"abc"}
	err = tagCommand(tagCmd, args)
	assert.ErrorContains(t, err, "invalid args for tag command")
}

func TestRuntag(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	args := []string{"abc", "abc"}
	err := runTag(ctx, &cli, args)
	assert.NilError(t, err)
}
