// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// iSula-Kits licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-06-04
// Description: This file is "logout" command for backend

package daemon

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
)

// Logout returns logout response
func (b *Backend) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	logrus.WithFields(logrus.Fields{
		"Server": req.GetServer(),
		"All":    req.GetAll(),
	}).Info("LogoutRequest received")

	if err := validLogoutOpts(req); err != nil {
		return &pb.LogoutResponse{Result: "Logout Failed"}, err
	}

	sysCtx := image.GetSystemContext()
	sysCtx.DockerCertPath = filepath.Join(constant.DefaultCertRoot, req.Server)

	if req.All {
		if err := config.RemoveAllAuthentication(sysCtx); err != nil {
			return &pb.LogoutResponse{Result: "Remove authentications failed"}, err
		}
		logrus.Info("Success logout from all servers")

		return &pb.LogoutResponse{Result: "Removed authentications"}, nil
	}

	err := config.RemoveAuthentication(sysCtx, req.Server)
	if err == nil {
		msg := fmt.Sprintf("Removed authentication for %s", req.Server)
		logrus.Infof("Success logout from server: %q", req.Server)
		return &pb.LogoutResponse{Result: msg}, nil
	}
	if strings.Contains(err.Error(), config.ErrNotLoggedIn.Error()) {
		msg := fmt.Sprintf("Not logged in registry %s", req.Server)
		return &pb.LogoutResponse{Result: msg}, config.ErrNotLoggedIn
	}
	msg := fmt.Sprintf("Logout for %s failed: %v", req.Server, err)
	return &pb.LogoutResponse{Result: msg}, err
}

func validLogoutOpts(req *pb.LogoutRequest) error {
	if req.All {
		return nil
	}

	if req.Server == "" {
		return errors.New(emptyServer)
	}

	return nil
}
