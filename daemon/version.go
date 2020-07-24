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
// Description: This file is "version" command for backend

package daemon

import (
	"context"
	"runtime"
	"strconv"
	"time"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/pkg/version"
)

// Version to get server version information
func (b *Backend) Version(ctx context.Context, req *gogotypes.Empty) (*pb.VersionResponse, error) {
	logrus.Info("VersionRequest received")

	var err error
	const base, baseSize = 10, 64
	buildTime := int64(0)
	if version.BuildInfo != "" {
		buildTime, err = strconv.ParseInt(version.BuildInfo, base, baseSize)
		if err != nil {
			return &pb.VersionResponse{}, err
		}
	}

	return &pb.VersionResponse{
		Version:   version.Version,
		GoVersion: runtime.Version(),
		GitCommit: version.GitCommit,
		BuildTime: time.Unix(buildTime, 0).Format(time.ANSIC),
		OsArch:    runtime.GOOS + "/" + runtime.GOARCH,
	}, nil
}
