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
// Description: This file is for grpc server test.

package daemon

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRunGrpc(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := prepare(t)
	defer tmpClean(d)

	d.Daemon.opts.Group = "isula"
	err := d.Daemon.NewGrpcServer()
	assert.NilError(t, err)

	errCh := make(chan error)
	err = d.Daemon.grpc.Run(ctx, errCh, cancel)
	assert.NilError(t, err)

}
