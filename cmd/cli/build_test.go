// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-01-20
// Description: This file is used for building test

package main

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"gotest.tools/assert"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	_ "isula.org/isula-build/exporter/register"
	"isula.org/isula-build/util"
)

type mockClient struct {
	client pb.ControlClient
}

func newMockClient(gcli *mockGrpcClient) mockClient {
	return mockClient{
		client: gcli,
	}
}

func (cli *mockClient) Client() pb.ControlClient {
	return cli.client
}

func (cli *mockClient) Close() error {
	return nil
}

type mockDaemon struct {
	buildReq  *pb.BuildRequest
	statusReq *pb.StatusRequest
	removeReq *pb.RemoveRequest
	loadReq   *pb.LoadRequest
	loginReq  *pb.LoginRequest
	logoutReq *pb.LogoutRequest
	importReq *pb.ImportRequest
}

func newMockDaemon() *mockDaemon {
	return &mockDaemon{}
}

func (f *mockDaemon) importImage(_ context.Context, opts ...grpc.CallOption) (pb.Control_ImportClient, error) {
	return &mockImportClient{}, nil
}

func (f *mockDaemon) load(_ context.Context, in *pb.LoadRequest, opts ...grpc.CallOption) (pb.Control_LoadClient, error) {
	f.loadReq = in
	return &mockLoadClient{}, nil
}

func (f *mockDaemon) build(_ context.Context, in *pb.BuildRequest, opts ...grpc.CallOption) (pb.Control_BuildClient, error) {
	f.buildReq = in
	return &mockBuildClient{}, nil
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

func (f *mockDaemon) login(_ context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	f.loginReq = in
	serverLen := len(f.loginReq.Server)
	if serverLen == 0 || serverLen > 128 {
		return &pb.LoginResponse{
			Content: "Login Failed",
		}, errors.New("empty server address")
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

func TestRunBuildWithLocalDockerfile(t *testing.T) {
	dockerfile := `
		FROM alpine:latest
		RUN echo hello world
		`
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile("Dockerfile", dockerfile))
	defer tmpDir.Remove()

	mockBuild := newMockDaemon()
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{imageBuildFunc: mockBuild.build})

	buildOpts.file = tmpDir.Join("Dockerfile")
	var args []string
	err := newBuildOptions(args)
	assert.NilError(t, err)
	buildOpts.output = "docker-daemon:isula:latest"
	_, err = runBuild(ctx, &cli)

	assert.NilError(t, err)
	assert.Equal(t, mockBuild.dockerfile(t), dockerfile)
	wd, err := os.Getwd()
	assert.NilError(t, err)
	realWd, err := filepath.EvalSymlinks(wd)
	assert.NilError(t, err)
	assert.Equal(t, mockBuild.contextDir(t), realWd)
}

func TestRunStatus(t *testing.T) {
	mockBuild := newMockDaemon()
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{statusFunc: mockBuild.status})
	var args []string
	err := newBuildOptions(args)
	assert.NilError(t, err)
	buildOpts.buildID = "09f5f06de97cdf53d5d94cbb6a11e61b646ae4685ed003a752ebde352f09175a"
	err = runStatus(ctx, &cli)
	assert.NilError(t, err)
}

func TestRunBuildWithDefaultDockerFile(t *testing.T) {
	dockerfile := `
		FROM alpine:latest
		RUN echo hello world
		`
	wd, err := os.Getwd()
	assert.NilError(t, err)
	realWd, err := filepath.EvalSymlinks(wd)
	assert.NilError(t, err)
	filePath := filepath.Join(realWd, "Dockerfile")
	err = ioutil.WriteFile(filePath, []byte(dockerfile), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	defer func() {
		err = os.Remove(filePath)
		assert.NilError(t, err)
	}()

	mockBuild := newMockDaemon()
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{imageBuildFunc: mockBuild.build})

	buildOpts.file = ""
	var args []string
	err = newBuildOptions(args)
	assert.NilError(t, err)
	buildOpts.output = "docker-daemon:isula:latest"
	_, err = runBuild(ctx, &cli)

	assert.NilError(t, err)
	assert.Equal(t, mockBuild.dockerfile(t), dockerfile)
	assert.Equal(t, mockBuild.contextDir(t), realWd)
}

// Test runBuild with non archive exporter
// case 1. docker-daemon exporter
// expect: pass
func TestRunBuildWithNArchiveExporter(t *testing.T) {
	type testcase struct {
		exporter string
		descSpec string
	}
	dockerfile := `
		FROM alpine:latest
		RUN echo hello world
		`
	wd, err := os.Getwd()
	assert.NilError(t, err)
	filePath := filepath.Join(wd, "Dockerfile")
	err = ioutil.WriteFile(filePath, []byte(dockerfile), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	defer func() {
		err = os.Remove(filePath)
		assert.NilError(t, err)
	}()

	mockBuild := newMockDaemon()
	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{imageBuildFunc: mockBuild.build})

	buildOpts.file = ""
	var args []string
	err = newBuildOptions(args)
	assert.NilError(t, err)

	var testcases = []testcase{
		{
			exporter: "docker-daeomn",
			descSpec: "docker-daemon:isula:latest",
		},
	}
	for _, tc := range testcases {
		buildOpts.output = tc.descSpec
		gotImageID, err := runBuild(ctx, &cli)
		assert.NilError(t, err)
		assert.Equal(t, gotImageID, imageID)
	}
}

// Test runBuild
// case 1. docker-archive exporter
// expect: pass
func TestRunBuildWithArchiveExporter(t *testing.T) {
	type testcase struct {
		exporter string
		descSpec string
	}
	dockerfile := `
		FROM alpine:latest
		RUN echo hello world
		`
	wd, err := os.Getwd()
	assert.NilError(t, err)
	filePath := filepath.Join(wd, "Dockerfile")
	err = ioutil.WriteFile(filePath, []byte(dockerfile), constant.DefaultSharedFileMode)
	assert.NilError(t, err)
	defer func() {
		err = os.Remove(filePath)
		assert.NilError(t, err)
	}()

	ctx := context.Background()
	cli := newMockClient(&mockGrpcClient{})

	buildOpts.file = ""
	var args []string
	err = newBuildOptions(args)
	assert.NilError(t, err)

	var testcases = []testcase{
		{
			exporter: "docker-archive",
			descSpec: "docker-archive:/tmp/image:isula:latest",
		},
	}
	for _, tc := range testcases {
		buildOpts.output = tc.descSpec
		gotImageID, err := runBuild(ctx, &cli)
		assert.NilError(t, err)
		assert.Equal(t, gotImageID, imageID)

		segments := strings.Split(tc.descSpec, ":")
		path := segments[1]
		_, err = os.Stat(path)
		assert.Assert(t, err == nil || os.IsExist(err))
		os.Remove(path)
	}
}

// Test readDockerfile
// case 1. file with full path
// expect: pass
func TestReadDockerfileWithFullpath(t *testing.T) {
	dockerfile := `
FROM alpine:latest
RUN echo hello world
`
	filename := "testDockerfile"
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile(filename, dockerfile))
	defer tmpDir.Remove()

	buildOpts.contextDir, _ = os.Getwd()
	buildOpts.file = tmpDir.Join(filename)

	_, err := readDockerfile()
	assert.NilError(t, err)
}

// Test readDockerfile
// case 2. file with only filename, will be resolved in contextDir
// expect: pass
func TestReadDockerfileWithFullname(t *testing.T) {
	dockerfile := `
FROM alpine:latest
RUN echo hello world
`
	filename := "testDockerfile"
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile(filename, dockerfile))
	defer tmpDir.Remove()

	buildOpts.contextDir = tmpDir.Path()
	buildOpts.file = "testDockerfile"

	_, err := readDockerfile()
	assert.NilError(t, err)
}

// Test readDockerfile
// case 3. file with no filename, will be resolved by contextDir+Dockerfile
// expect: pass
func TestReadDockerfileWithNoName(t *testing.T) {
	dockerfile := `
FROM alpine:latest
RUN echo hello world
`
	filename := "Dockerfile"
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile(filename, dockerfile))
	defer tmpDir.Remove()

	buildOpts.contextDir = tmpDir.Path()
	buildOpts.file = ""

	_, err := readDockerfile()
	assert.NilError(t, err)
}

// Test readDockerfile
// case 4. file with no content
// expect: return error
func TestReadDockerfileWithNoContent(t *testing.T) {
	dockerfile := ``

	filename := "Dockerfile"
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile(filename, dockerfile))
	defer tmpDir.Remove()

	buildOpts.contextDir = tmpDir.Path()
	buildOpts.file = filename

	_, err := readDockerfile()
	assert.ErrorContains(t, err, "file is empty")
}

// Test readDockerfile
// case 5. file with "directory"
// expect: return error
func TestReadDockerfileWithDirectory(t *testing.T) {
	buildOpts.contextDir = ""
	buildOpts.file = "."

	_, err := readDockerfile()
	assert.ErrorContains(t, err, "should be a regular file")
}

func TestNewBuildOptions(t *testing.T) {
	// no args case use current working directory as context directory
	cwd, err := os.Getwd()
	realCwd, err := filepath.EvalSymlinks(cwd)
	assert.NilError(t, err)
	var args []string
	err = newBuildOptions(args)
	assert.NilError(t, err)
	assert.Equal(t, buildOpts.contextDir, realCwd)

	// normal case
	args = []string{".", "abc"}
	absPath, err := filepath.Abs(".")
	realPath, err := filepath.EvalSymlinks(absPath)
	assert.NilError(t, err)
	err = newBuildOptions(args)
	assert.NilError(t, err)
	assert.Equal(t, buildOpts.contextDir, realPath)

	// context directory not exist
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	args = []string{tmpDir.Path() + "/test"}
	err = newBuildOptions(args)
	assert.ErrorContains(t, err, "error getting the real path")

	// context directory is not a directory
	err = ioutil.WriteFile(tmpDir.Path()+"/test", []byte(""), constant.DefaultRootFileMode)
	args = []string{tmpDir.Path() + "/test"}
	err = newBuildOptions(args)
	assert.ErrorContains(t, err, "should be a directory")
}

func TestCheckAndProcessOut(t *testing.T) {
	type testcase struct {
		name     string
		output   string
		expect   string
		errStr   string
		isIsulad bool
		isErr    bool
	}

	testcases := []testcase{
		{
			name:     "docker-archive",
			output:   "docker-archive:/root/docker-archive.tar",
			expect:   "/root/docker-archive.tar",
			isIsulad: false,
		},
		{
			name:     "docker-daemon",
			output:   "docker-daemon:busybox:latest",
			expect:   "",
			isIsulad: false,
		},
		{
			name:     "docker-registry",
			output:   "docker://registry.example.com/busybox:latest",
			expect:   "",
			isIsulad: false,
		},
		{
			name:     "empyty exporter",
			output:   "",
			expect:   "",
			isIsulad: false,
		},
		{
			name:     "only has colon",
			output:   ":",
			expect:   "",
			isIsulad: false,
			errStr:   "transport should not be empty",
			isErr:    true,
		},
		{
			name:     "only has transport",
			output:   "docker-archive:",
			expect:   "",
			isIsulad: false,
			errStr:   "destination should not be empty",
			isErr:    true,
		},
		{
			name:     "invalid exporter with no dest1",
			output:   "docker-archive",
			expect:   "",
			isErr:    true,
			errStr:   "destination should not be empty",
			isIsulad: false,
		},
		{
			name:     "invalid exporter with no dest3",
			output:   "docker-archive:  ",
			expect:   "",
			isErr:    true,
			errStr:   "destination should not be empty",
			isIsulad: false,
		},
		{
			name:     "invalid exporter with no dest2",
			output:   "docker-archive:",
			expect:   "",
			isErr:    true,
			errStr:   "destination should not be empty",
			isIsulad: false,
		},
		{
			name:     "invalid exporter with no transport",
			output:   ":/test/images",
			expect:   "",
			isErr:    true,
			errStr:   "transport should not be empty",
			isIsulad: false,
		},
		{
			name:     "invalid transport",
			output:   "docker-isula:/root/docker-isula.tar",
			expect:   "/root/docker-isula.tar",
			errStr:   "not support",
			isErr:    true,
			isIsulad: false,
		},
		{
			name:   "invalid docker transport longer than limit",
			output: "docker:lcoalhostaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:5000/isula:test",
			errStr: "output should not longer than",
			isErr:  true,
		},
		{
			name:   "invalid docker-daemon transport longer than limit",
			output: "docker-daemon:isulaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:test",
			errStr: "output should not longer than",
			isErr:  true,
		},
		{
			name:   "invalid docker-archive transport longer than limit",
			output: "docker-archive:isula.tar:isulaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:test",
			errStr: "output should not longer than",
			isErr:  true,
		},
		{
			name:   "invalid isulad transport longer than limit",
			output: "isulad:isulaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:test",
			errStr: "output should not longer than",
			isErr:  true,
		},
		{
			name:   "invalid isulad transport",
			output: "isulad:isula",
			errStr: "invalid isulad output format",
			isErr:  true,
		},
		{
			name:   "invalid isulad transport",
			output: "isulad:isula",
			errStr: "invalid isulad output format",
			isErr:  true,
		},
		{
			name:   "invalid isulad transport 2",
			output: "isulad:isula:isula:isula",
			errStr: "invalid isulad output format",
			isErr:  true,
		},
		{
			name:     "valid isulad transport",
			output:   "isulad:isula:latest",
			expect:   "/var/tmp/isula-build-tmp-abc123.tar",
			isErr:    false,
			isIsulad: true,
		},
	}

	for _, tc := range testcases {
		buildOpts.buildID = "abc123"
		buildOpts.output = tc.output
		dest, isIsulad, err := checkAndProcessOutput()
		if tc.isErr {
			assert.ErrorContains(t, err, tc.errStr, tc.name)
		} else {
			assert.NilError(t, err)
			assert.Equal(t, dest, tc.expect, tc.name)
			assert.Equal(t, isIsulad, tc.isIsulad, tc.name)
		}
	}
}

func TestEncryptBuildArgs(t *testing.T) {
	var tests = []struct {
		name    string
		args    []string
		encrypt bool
		err     bool
	}{
		{
			name:    "case 1 - no build-args",
			args:    []string{},
			encrypt: false,
			err:     false,
		},
		{
			name:    "case 2 - normal build-args",
			args:    []string{"foo=bar", "testArg=arg"},
			encrypt: false,
			err:     false,
		},
		{
			name:    "case 3 - sensitive build-args",
			args:    []string{"foo=bar", "http_proxy=http://username:password@url.com/"},
			encrypt: true,
			err:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildOpts.buildArgs = tt.args
			buildOpts.encryptKey = ""

			if err := encryptBuildArgs(); (err == nil) != (!tt.err) {
				t.FailNow()
			}
			if tt.encrypt {
				for i := 0; i < len(tt.args); i++ {
					arg, err := util.DecryptAES(buildOpts.buildArgs[i], buildOpts.encryptKey)
					assert.NilError(t, err)
					assert.Equal(t, tt.args[i], arg)
				}
			} else {
				for i := 0; i < len(tt.args); i++ {
					assert.Equal(t, tt.args[i], buildOpts.buildArgs[i])
				}
			}

		})
	}
}
