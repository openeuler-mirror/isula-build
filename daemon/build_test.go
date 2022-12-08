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

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

func TestBuild(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	_, err := d.Daemon.backend.Build(context.Background(), &pb.BuildRequest{})
	assert.ErrorContains(t, err, "is not supported")

	_, err = d.Daemon.backend.Build(context.Background(), &pb.BuildRequest{BuildType: "ctr-img"})
	assert.ErrorContains(t, err, "wrong image format provided")

	buildID := util.GenerateNonCryptoID()[:constant.DefaultIDLen]
	go func() {
		<-d.Daemon.backend.syncBuildStatus(buildID)
	}()
	_, err = d.Daemon.backend.Build(context.Background(),
		&pb.BuildRequest{BuildType: "ctr-img", Format: "oci", BuildID: buildID})
	assert.ErrorContains(t, err, "parse dockerfile failed")
}
