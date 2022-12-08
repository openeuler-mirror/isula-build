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
// Description: This file is for info test.

package daemon

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"

	pb "isula.org/isula-build/api/services"
)

func TestInfo(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	_, err := d.Daemon.backend.Info(context.Background(), &pb.InfoRequest{})
	assert.NilError(t, err)
	_, err = d.Daemon.backend.Info(context.Background(), &pb.InfoRequest{Verbose: true})
	assert.NilError(t, err)
}

func TestGetRegistryInfo(t *testing.T) {
	_, _, _, err := getRegistryInfo()
	assert.NilError(t, err)
}
