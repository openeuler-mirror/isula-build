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
// Description: This file is used for client testing

package main

import (
	"context"
	"io"

	types "github.com/gogo/protobuf/types"
	"google.golang.org/grpc"

	pb "isula.org/isula-build/api/services"
)

const (
	imageID = "38b993607bcabe01df1dffdf01b329005c6a10a36d557f9d073fc25943840c66"
	content = `STEP 1: FROM busybox:latest
				STEP 2: RUN echo hello world`
)

type mockGrpcClient struct {
	imageBuildFunc  func(ctx context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (pb.Control_BuildClient, error)
	removeFunc      func(ctx context.Context, in *pb.RemoveRequest, opts ...grpc.CallOption) (pb.Control_RemoveClient, error)
	listFunc        func(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error)
	statusFunc      func(ctx context.Context, in *pb.StatusRequest, opts ...grpc.CallOption) (pb.Control_StatusClient, error)
	healthCheckFunc func(ctx context.Context, in *types.Empty, opts ...grpc.CallOption) (*pb.HealthCheckResponse, error)
	loginFunc       func(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error)
	logoutFunc      func(ctx context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error)
	loadFunc        func(ctx context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (*pb.LoadResponse, error)
}

func (gcli *mockGrpcClient) Build(ctx context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (pb.Control_BuildClient, error) {
	if gcli.imageBuildFunc != nil {
		return gcli.imageBuildFunc(ctx, in, opts...)
	}
	return &mockBuildClient{isArchive: true}, nil
}

func (gcli *mockGrpcClient) Remove(ctx context.Context, in *pb.RemoveRequest, opts ...grpc.CallOption) (pb.Control_RemoveClient, error) {
	if gcli.removeFunc != nil {
		return gcli.removeFunc(ctx, in, opts...)
	}
	return &mockRemoveClient{}, nil
}

func (gcli *mockGrpcClient) List(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error) {
	return &pb.ListResponse{
		Images: []*pb.ListResponse_ImageInfo{{
			Repository: "repository",
			Tag:        "tag",
			Id:         "abcdefg1234567",
			Created:    "2020-01-01",
			Size_:      "100 MB",
		}},
	}, nil
}

func (gcli *mockGrpcClient) Version(ctx context.Context, in *types.Empty, opts ...grpc.CallOption) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:   "",
		GoVersion: "",
		GitCommit: "",
		BuildTime: "",
		OsArch:    "",
	}, nil
}

func (gcli *mockGrpcClient) Status(ctx context.Context, in *pb.StatusRequest, opts ...grpc.CallOption) (pb.Control_StatusClient, error) {
	if gcli.statusFunc != nil {
		return gcli.statusFunc(ctx, in, opts...)
	}
	return &mockStatusClient{}, nil
}

func (gcli *mockGrpcClient) HealthCheck(ctx context.Context, in *types.Empty, opts ...grpc.CallOption) (*pb.HealthCheckResponse, error) {
	if gcli.healthCheckFunc != nil {
		return gcli.healthCheckFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (gcli *mockGrpcClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	if gcli.loginFunc != nil {
		return gcli.loginFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (gcli *mockGrpcClient) Logout(ctx context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error) {
	if gcli.loginFunc != nil {
		return gcli.logoutFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (gcli *mockGrpcClient) Load(ctx context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (*pb.LoadResponse, error) {
	if gcli.loadFunc != nil {
		return gcli.loadFunc(ctx, in, opts...)
	}
	return nil, nil
}

type mockBuildClient struct {
	grpc.ClientStream
	isArchive bool
}

type mockStatusClient struct {
	grpc.ClientStream
}

type mockRemoveClient struct {
	grpc.ClientStream
}

func (bcli *mockBuildClient) Recv() (*pb.BuildResponse, error) {
	resp := &pb.BuildResponse{
		ImageID: imageID,
		Data:    []byte{},
	}
	if bcli.isArchive {
		return resp, io.EOF
	}
	return resp, nil
}

func (scli *mockStatusClient) Recv() (*pb.StatusResponse, error) {
	resp := &pb.StatusResponse{
		Content: content,
	}
	return resp, io.EOF
}

func (rcli *mockRemoveClient) Recv() (*pb.RemoveResponse, error) {
	resp := &pb.RemoveResponse{
		LayerMessage: imageID,
	}
	return resp, io.EOF
}
