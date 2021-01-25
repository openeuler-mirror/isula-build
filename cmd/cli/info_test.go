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
// Create: 2020-08-03
// Description: This file is used for testing command info

package main

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	pb "isula.org/isula-build/api/services"
)

func TestInfoCommand(t *testing.T) {
	infoCmd := NewInfoCmd()
	var args []string
	err := infoCommand(infoCmd, args)
	assert.ErrorContains(t, err, "isula_build.sock")
}

func TestGetInfoFromDaemon(t *testing.T) {
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})
	err := runInfo(ctx, &cli)
	assert.NilError(t, err)
}

func TestPrintInfo(t *testing.T) {
	infoData := &pb.InfoResponse{
		MemInfo: &pb.MemData{
			MemTotal:  123,
			MemFree:   123,
			SwapTotal: 123,
			SwapFree:  123,
		},
		StorageInfo: &pb.StorageData{
			StorageDriver:    "overlay",
			StorageBackingFs: "extfs",
		},
		RegistryInfo: &pb.RegistryData{
			RegistriesSearch:   []string{"docker.io"},
			RegistriesInsecure: []string{"localhost:5000"},
			RegistriesBlock:    nil,
		},
		DataRoot:     "/var/lib/isula-build/",
		RunRoot:      "/var/run/isula-build/",
		OCIRuntime:   "runc",
		BuilderNum:   0,
		GoRoutines:   10,
		Experimental: false,
	}
	printInfo(infoData)
}
