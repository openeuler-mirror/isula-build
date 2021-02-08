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
// Create: 2020-06-02
// Description: This file is "login" command for backend

package daemon

import (
	"context"
	"crypto"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/util"
)

const (
	loginSuccess             = "Login Succeeded"
	loginFailed              = "Login Failed"
	loginUnauthorized        = "Unauthorized login attempt"
	loginSetAuthFailed       = "Set Auth Failed"
	emptyServer              = "empty server address"
	emptyAuth                = "empty auth info"
	errTryToUseAuth          = "Failed to authenticate existing credentials, try to use auth directly"
	loginSuccessWithAuthFile = "Login Succeed with AuthFile"
)

// Login returns login response
func (b *Backend) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	logrus.WithFields(logrus.Fields{
		"Server":   req.GetServer(),
		"Username": req.GetUsername(),
	}).Info("LoginRequest received")

	err := validLoginOpts(req)
	if err != nil {
		return &pb.LoginResponse{Content: loginFailed}, err
	}

	sysCtx := image.GetSystemContext()
	sysCtx.DockerCertPath, err = securejoin.SecureJoin(constant.DefaultCertRoot, req.Server)
	if err != nil {
		return &pb.LoginResponse{Content: loginFailed}, err
	}

	if loginWithAuthFile(req) {
		auth, gErr := config.GetCredentials(sysCtx, req.Server)
		if gErr != nil || auth.Password == "" {
			auth = types.DockerAuthConfig{}
			return &pb.LoginResponse{Content: errTryToUseAuth}, errors.Errorf("failed to read auth file: %v", errTryToUseAuth)
		}

		usernameFromAuth, passwordFromAuth := auth.Username, auth.Password
		// use existing credentials from authFile if any
		if usernameFromAuth != "" && passwordFromAuth != "" {
			logrus.Infof("Authenticating with existing credentials")
			err = docker.CheckAuth(ctx, sysCtx, usernameFromAuth, passwordFromAuth, req.Server)
			if err == nil {
				logrus.Infof("Success login server: %s by auth file with username: %s", req.Server, usernameFromAuth)
				return &pb.LoginResponse{Content: loginSuccessWithAuthFile}, nil
			}
			return &pb.LoginResponse{Content: errTryToUseAuth}, errors.Wrap(err, errTryToUseAuth)
		}
	}

	// use username and password from client to access
	password, err := util.DecryptRSA(req.Password, b.daemon.key, crypto.SHA512)
	if err != nil {
		return &pb.LoginResponse{Content: err.Error()}, err
	}

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

func loginWithAuthFile(req *pb.LoginRequest) bool {
	if req.Password == "" && req.Username == "" && req.Server != "" {
		return true
	}
	return false
}

func validLoginOpts(req *pb.LoginRequest) error {
	// in this scenario, the client just send server address to backend,
	// there is no pass and user name info sent to here.
	// we just check if there is valid auth info in auth.json later.
	if req.Password == "" && req.Username == "" && req.Server != "" {
		return nil
	}
	if req.Server == "" {
		return errors.New(emptyServer)
	}

	if req.Password == "" {
		return errors.New(emptyAuth)
	}

	return nil
}
