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
// Create: 2020-08-12
// Description: This file is used for mock test

package main

import (
	"context"
	"io"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "isula.org/isula-build/api/services"
)

const (
	imageID = "38b993607bcabe01df1dffdf01b329005c6a10a36d557f9d073fc25943840c66"
	content = `STEP 1: FROM busybox:latest
				STEP 2: RUN echo hello world`
)

type mockClient struct {
	client pb.ControlClient
}

type mockDaemon struct {
	buildReq  *pb.BuildRequest
	statusReq *pb.StatusRequest
	removeReq *pb.RemoveRequest
	loadReq   *pb.LoadRequest
	loginReq  *pb.LoginRequest
	logoutReq *pb.LogoutRequest
	pushReq   *pb.PushRequest
	pullReq   *pb.PullRequest
	importReq *pb.ImportRequest // nolint
	saveReq   *pb.SaveRequest   // nolint
}

type mockImportClient struct {
	grpc.ClientStream
}

type mockStatusClient struct {
	grpc.ClientStream
}

type mockPushClient struct {
	grpc.ClientStream
}

type mockPullClient struct {
	grpc.ClientStream
}

type mockRemoveClient struct {
	grpc.ClientStream
}

type mockLoadClient struct {
	grpc.ClientStream
}

type mockSaveClient struct {
	grpc.ClientStream
}

type mockGrpcClient struct {
	imageBuildFunc  func(ctx context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (*pb.BuildResponse, error)
	removeFunc      func(ctx context.Context, in *pb.RemoveRequest, opts ...grpc.CallOption) (pb.Control_RemoveClient, error)
	listFunc        func(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error) // nolint
	statusFunc      func(ctx context.Context, in *pb.StatusRequest, opts ...grpc.CallOption) (pb.Control_StatusClient, error)
	healthCheckFunc func(ctx context.Context, in *types.Empty, opts ...grpc.CallOption) (*pb.HealthCheckResponse, error)
	loginFunc       func(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error)
	logoutFunc      func(ctx context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error)
	loadFunc        func(ctx context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (pb.Control_LoadClient, error)
	pushFunc        func(ctx context.Context, in *pb.PushRequest, opts ...grpc.CallOption) (pb.Control_PushClient, error)
	pullFunc        func(ctx context.Context, in *pb.PullRequest, opts ...grpc.CallOption) (pb.Control_PullClient, error)
	importFunc      func(ctx context.Context, in *pb.ImportRequest, opts ...grpc.CallOption) (pb.Control_ImportClient, error)
	saveFunc        func(ctx context.Context, in *pb.SaveRequest, opts ...grpc.CallOption) (pb.Control_SaveClient, error)
}

func newMockClient(gcli *mockGrpcClient) mockClient { // nolint
	return mockClient{
		client: gcli,
	}
}

func newMockDaemon() *mockDaemon { // nolint
	return &mockDaemon{}
}

func (gcli *mockGrpcClient) Build(ctx context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (*pb.BuildResponse, error) {
	if gcli.imageBuildFunc != nil {
		return gcli.imageBuildFunc(ctx, in, opts...)
	}
	return &pb.BuildResponse{ImageID: imageID}, nil
}

func (gcli *mockGrpcClient) Import(ctx context.Context, in *pb.ImportRequest, opts ...grpc.CallOption) (pb.Control_ImportClient, error) {
	if gcli.importFunc != nil {
		return gcli.importFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (gcli *mockGrpcClient) Remove(ctx context.Context, in *pb.RemoveRequest, opts ...grpc.CallOption) (pb.Control_RemoveClient, error) {
	if gcli.removeFunc != nil {
		return gcli.removeFunc(ctx, in, opts...)
	}
	return &mockRemoveClient{}, nil
}

func (gcli *mockGrpcClient) Save(ctx context.Context, in *pb.SaveRequest, opts ...grpc.CallOption) (pb.Control_SaveClient, error) {
	if gcli.saveFunc != nil {
		return gcli.saveFunc(ctx, in, opts...)
	}
	return &mockSaveClient{}, nil
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

func (gcli *mockGrpcClient) Info(ctx context.Context, in *pb.InfoRequest, opts ...grpc.CallOption) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{
		MemInfo: &pb.MemData{
			MemTotal:  123,
			MemFree:   123,
			SwapTotal: 123,
			SwapFree:  123,
		},
		StorageInfo: &pb.StorageData{
			StorageDriver:    "mockDriver",
			StorageBackingFs: "mockBackingFs",
		},
		RegistryInfo: &pb.RegistryData{
			RegistriesSearch:   []string{"mockSearch"},
			RegistriesInsecure: []string{"mockInsecure"},
			RegistriesBlock:    nil,
		},
		DataRoot:   "/mock/data/root",
		RunRoot:    "/mock/run/root",
		OCIRuntime: "mockRuntime",
		BuilderNum: 0,
		GoRoutines: 0,
		MemStat:    nil,
	}, nil
}

func (gcli *mockGrpcClient) Tag(ctx context.Context, in *pb.TagRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return &types.Empty{}, nil
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
	if gcli.logoutFunc != nil {
		return gcli.logoutFunc(ctx, in, opts...)
	}
	return &pb.LogoutResponse{Result: "Success Logout"}, nil
}

func (gcli *mockGrpcClient) Push(ctx context.Context, in *pb.PushRequest, opts ...grpc.CallOption) (pb.Control_PushClient, error) {
	if gcli.pushFunc != nil {
		return gcli.pushFunc(ctx, in, opts...)
	}
	return &mockPushClient{}, nil
}

func (gcli *mockGrpcClient) Pull(ctx context.Context, in *pb.PullRequest, opts ...grpc.CallOption) (pb.Control_PullClient, error) {
	if gcli.pullFunc != nil {
		return gcli.pullFunc(ctx, in, opts...)
	}
	return &mockPullClient{}, nil
}

func (gcli *mockGrpcClient) Load(ctx context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (pb.Control_LoadClient, error) {
	if gcli.loadFunc != nil {
		return gcli.loadFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (icli *mockImportClient) Recv() (*pb.ImportResponse, error) {
	resp := &pb.ImportResponse{
		Log: "Import success with image id: " + imageID,
	}
	return resp, nil
}

func (icli *mockImportClient) Send(*pb.ImportRequest) error {
	return nil
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

func (lcli *mockLoadClient) Recv() (*pb.LoadResponse, error) {
	resp := &pb.LoadResponse{
		Log: "Getting image source signatures",
	}
	return resp, io.EOF
}

func (m mockPushClient) Recv() (*pb.PushResponse, error) {
	return &pb.PushResponse{}, io.EOF
}

func (m mockPullClient) Recv() (*pb.PullResponse, error) {
	return &pb.PullResponse{}, io.EOF
}

func (scli *mockSaveClient) Recv() (*pb.SaveResponse, error) {
	return &pb.SaveResponse{}, io.EOF
}

func (cli *mockClient) Client() pb.ControlClient {
	return cli.client
}

func (cli *mockClient) Close() error {
	return nil
}

func (f *mockDaemon) importImage(_ context.Context, opts ...grpc.CallOption) (pb.Control_ImportClient, error) {
	return &mockImportClient{}, nil
}

func (f *mockDaemon) load(_ context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (pb.Control_LoadClient, error) {
	f.loadReq = in
	return &mockLoadClient{}, nil
}

func (f *mockDaemon) build(_ context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (*pb.BuildResponse, error) {
	f.buildReq = in
	return &pb.BuildResponse{ImageID: imageID}, nil
}

func (f *mockDaemon) status(_ context.Context, in *pb.StatusRequest, opts ...grpc.CallOption) (pb.Control_StatusClient, error) {
	f.statusReq = in
	return &mockStatusClient{}, nil
}

func (f *mockDaemon) dockerfile(t *testing.T) string {
	t.Helper()
	return f.buildReq.FileContent
}

func (f *mockDaemon) contextDir(t *testing.T) string {
	t.Helper()
	return f.buildReq.ContextDir
}

func (f *mockDaemon) remove(_ context.Context, in *pb.RemoveRequest, opts ...grpc.CallOption) (pb.Control_RemoveClient, error) {
	f.removeReq = in
	return &mockRemoveClient{}, nil
}

func (f *mockDaemon) push(_ context.Context, in *pb.PushRequest, opts ...grpc.CallOption) (pb.Control_PushClient, error) {
	f.pushReq = in
	return &mockPushClient{}, nil
}

func (f *mockDaemon) pull(_ context.Context, in *pb.PullRequest, opts ...grpc.CallOption) (pb.Control_PullClient, error) {
	f.pullReq = in
	return &mockPullClient{}, nil
}

func (f *mockDaemon) save(_ context.Context, in *pb.SaveRequest, opts ...grpc.CallOption) (pb.Control_SaveClient, error) {
	f.saveReq = in
	imageList := f.saveReq.Images
	// construct failed env when trying to save image id "38b993607bcabe01df1dffdf01b329005c6a10a36d557f9d073fc25943840c66"
	for _, id := range imageList {
		if id == imageID {
			return &mockSaveClient{}, errors.Errorf("failed to save image %s", imageID)
		}
	}

	return &mockSaveClient{}, nil
}

func (f *mockDaemon) login(_ context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	f.loginReq = in
	username := f.loginReq.Username
	password := f.loginReq.Password
	server := f.loginReq.Server
	serverLen := len(server)
	if serverLen == 0 || serverLen > 128 {
		return &pb.LoginResponse{
			Content: "Login Failed",
		}, errors.New("empty server address")
	}
	if username == "" && password == "" && server != "" {
		return &pb.LoginResponse{
			Content: "Failed to authenticate existing credentials",
		}, errors.New("Failed to authenticate existing credentials")
	}

	return &pb.LoginResponse{Content: "Success"}, nil
}

func (f *mockDaemon) logout(_ context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error) {
	f.logoutReq = in
	serverLen := len(f.logoutReq.Server)
	if serverLen == 0 || serverLen > 128 {
		return &pb.LogoutResponse{Result: "Logout Failed"}, errors.New("empty server address")
	}

	return &pb.LogoutResponse{Result: "Success"}, nil
}
