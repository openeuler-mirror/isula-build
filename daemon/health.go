// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-01-20
// Description: This file is used to check healthy between client and server

package daemon

import (
	"context"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
)

// HealthCheck returns daemon healthy condition
func (b *Backend) HealthCheck(ctx context.Context, req *gogotypes.Empty) (*pb.HealthCheckResponse, error) {
	logrus.Info("HealthCheckRequest received")

	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}
