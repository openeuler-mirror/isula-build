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
// Description: This is test file for version

package daemon

import (
	"context"
	"testing"

	gogotypes "github.com/gogo/protobuf/types"
	"gotest.tools/assert"

	"isula.org/isula-build/pkg/version"
)

func TestVersion(t *testing.T) {
	backend := Backend{}
	_, err := backend.Version(context.Background(), &gogotypes.Empty{})
	assert.NilError(t, err)
}

func TestVersionParseFail(t *testing.T) {
	backend := Backend{}
	version.BuildInfo = "abc"
	_, err := backend.Version(context.Background(), &gogotypes.Empty{})
	assert.ErrorContains(t, err, "invalid syntax")
}
