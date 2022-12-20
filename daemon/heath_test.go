// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: sicheng Dai
// Create: 2020-01-20
// Description: This is test file for health

package daemon

import (
	"context"
	"testing"

	gogotypes "github.com/gogo/protobuf/types"
	"gotest.tools/v3/assert"
)

func TestHealthCheck(t *testing.T) {
	backend := Backend{}
	_, err := backend.HealthCheck(context.Background(), &gogotypes.Empty{})
	assert.NilError(t, err)
}
