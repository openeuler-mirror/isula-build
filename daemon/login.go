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
// Create: 2020-06-02
// Description: This file is "login" command for backend

package daemon

import (
	"context"
	"path/filepath"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/util"
)

const (
	loginSuccess       = "Login Succeeded"
	loginWithAuthFile  = "Login Succeed with AuthFile"
	loginFailed        = "Login Failed"
	loginUnauthorized  = "Unauthorized login attempt"
	loginSetAuthFailed = "Set Auth Failed"
	emptyKey           = "empty key found"
	emptyServer        = "empty server address"
	emptyAuth          = "empty auth info"
)

// Login returns login response
func (b *Backend) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	logrus.WithFields(logrus.Fields{
		"Server":   req.GetServer(),
		"Username": req.GetUsername(),
	}).Info("LoginRequest received")

	if err := validLoginOpts(req); err != nil {
		return &pb.LoginResponse{Content: loginFailed}, err
	}

	password, err := util.DecryptAES(req.Password, req.Key)
	if err != nil {
		return &pb.LoginResponse{Content: err.Error()}, err
	}

	sysCtx := image.GetSystemContext()
	sysCtx.DockerCertPath = filepath.Join(constant.DefaultCertRoot, req.Server)

	auth, err := config.GetCredentials(sysCtx, req.Server)
	if err != nil {
		auth = types.DockerAuthConfig{}
		return &pb.LoginResponse{Content: err.Error()}, errors.Wrapf(err, "failed to read auth file %v", image.DefaultAuthFile)
	}

	usernameFromAuth, passwordFromAuth := auth.Username, auth.Password
	// use existing credentials from authFile if any
	if usernameFromAuth != "" && passwordFromAuth != "" {
		logrus.Infof("Authenticating with existing credentials")
		err = docker.CheckAuth(ctx, sysCtx, usernameFromAuth, passwordFromAuth, req.Server)
		if err == nil {
			logrus.Infof("Success login server: %s by auth file with username: %s", req.Server, usernameFromAuth)
			return &pb.LoginResponse{Content: loginWithAuthFile}, err
		}
		logrus.Infof("Failed to authenticate existing credentials, try to use auth directly")
	}

	// use username and password from client to access
	if err = docker.CheckAuth(ctx, sysCtx, req.Username, password, req.Server); err != nil {
		// check if user is authorized
		if _, ok := err.(docker.ErrUnauthorizedForCredentials); ok {
			return &pb.LoginResponse{Content: loginUnauthorized}, err
		}
		return &pb.LoginResponse{Content: loginFailed}, err
	}

	if err = config.SetAuthentication(sysCtx, req.Server, req.Username, password); err != nil {
		return &pb.LoginResponse{Content: loginSetAuthFailed}, err
	}

	logrus.Infof("Success login server: %s with username: %s", req.Server, req.Username)

	return &pb.LoginResponse{Content: loginSuccess}, nil
}

func validLoginOpts(req *pb.LoginRequest) error {
	if req.Key == "" {
		return errors.New(emptyKey)
	}
	if req.Server == "" {
		return errors.New(emptyServer)
	}

	if req.Password == "" {
		return errors.New(emptyAuth)
	}

	return nil
}
