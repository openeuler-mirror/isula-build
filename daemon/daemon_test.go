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
// Create: 2022-11-29
// Description: This file is for build test.

package daemon

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
)

func TestNewDaemon(t *testing.T) {
	_, err := NewDaemon(Options{}, nil)
	assert.NilError(t, err)
}

func TestRun(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	err := d.Daemon.Run()
	assert.ErrorContains(t, err, "create new GRPC socket failed")
	d.Daemon.Cleanup()
}

func TestNewBuilder(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	_, err := d.Daemon.NewBuilder(context.Background(), &pb.BuildRequest{BuildType: "ctr-img", Format: "oci"})
	assert.NilError(t, err)
}
