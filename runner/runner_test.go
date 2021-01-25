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
// Create: 2020-04-01
// Description: runner related functions tests

package runner

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
	"isula.org/isula-build/pkg/logger"
)

type mockRunc struct {
	Command       string
	Root          string
	Debug         bool
	Log           string
	LogFormat     string
	PdeathSignal  syscall.Signal
	Setpgid       bool
	Criu          string
	SystemdCgroup bool
	Rootless      *bool
}

func (r *mockRunc) Run(context context.Context, id, bundle string, opts *runc.CreateOpts) (int, error) {
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		return 1, errors.New("cmd.start failed")
	}
	pid := fmt.Sprintf("%d\n", cmd.Process.Pid)
	if err := ioutil.WriteFile(bundle+"/pid", []byte(pid), constant.DefaultSharedFileMode); err != nil {
		return 1, errors.New("write pid to file failed")
	}
	return 0, nil
}

func (r *mockRunc) Kill(context context.Context, id string, sig int, opts *runc.KillOpts) error {
	return nil
}

func (r *mockRunc) Delete(context context.Context, id string, opts *runc.DeleteOpts) error {
	return nil
}

func (r *mockRunc) State(context context.Context, id string) (*runc.Container, error) {
	return &runc.Container{
		ID:          "",
		Pid:         0,
		Status:      "stopped",
		Bundle:      "",
		Rootfs:      "",
		Created:     time.Time{},
		Annotations: nil,
	}, nil
}

type mockFailRunc struct {
	Command       string
	Root          string
	Debug         bool
	Log           string
	LogFormat     string
	PdeathSignal  syscall.Signal
	Setpgid       bool
	Criu          string
	SystemdCgroup bool
	Rootless      *bool
}

func (r *mockFailRunc) Run(context context.Context, id, bundle string, opts *runc.CreateOpts) (int, error) {
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		return 1, errors.New("cmd.start failed")
	}
	pid := fmt.Sprintf("%d\n", cmd.Process.Pid)
	if err := ioutil.WriteFile(bundle+"/pid", []byte(pid), constant.DefaultSharedFileMode); err != nil {
		return 1, errors.New("write pid to file failed")
	}
	return 1, errors.New("run error")
}

func (r *mockFailRunc) Kill(context context.Context, id string, sig int, opts *runc.KillOpts) error {
	return errors.New("kill error")
}

func (r *mockFailRunc) Delete(context context.Context, id string, opts *runc.DeleteOpts) error {
	return errors.New("delete error")
}

func (r *mockFailRunc) State(context context.Context, id string) (*runc.Container, error) {
	return &runc.Container{
		ID:          "",
		Pid:         0,
		Status:      "stopped",
		Bundle:      "",
		Rootfs:      "",
		Created:     time.Time{},
		Annotations: nil,
	}, errors.New("state error")
}

func TestRunnerRun(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	bundlePath := tmpDir.Path()
	spec := &specs.Spec{}
	cliLog := logger.NewCliLogger(constant.CliLogBufferLen)

	runner := NewOCIRunner(&OCIRunOpts{
		Ctx:         context.Background(),
		Spec:        spec,
		RuntimePath: "/aaaaa",
		BundlePath:  bundlePath,
		NoPivot:     false,
		Output:      cliLog,
	})
	err := runner.Run()
	assert.ErrorContains(t, err, "runOCIRuntime err")
}

func TestRunOCIRuntimeSucceed(t *testing.T) {
	runtime := &mockRunc{
		Command:       "",
		Root:          "",
		Debug:         false,
		Log:           "",
		LogFormat:     "",
		PdeathSignal:  0,
		Setpgid:       false,
		Criu:          "",
		SystemdCgroup: false,
		Rootless:      nil,
	}

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	bundlePath := tmpDir.Path()
	cliLog := logger.NewCliLogger(constant.CliLogBufferLen)
	spec := &specs.Spec{}

	runner := NewOCIRunner(&OCIRunOpts{
		Ctx:         context.Background(),
		Spec:        spec,
		RuntimePath: "",
		BundlePath:  bundlePath,
		NoPivot:     false,
		Output:      cliLog,
	})
	runner.runtime = runtime
	_, err := runner.runContainer()
	assert.NilError(t, err)
}

func TestRunOCIRuntimeContextCancel(t *testing.T) {
	runtime := &mockRunc{
		Command:       "",
		Root:          "",
		Debug:         false,
		Log:           "",
		LogFormat:     "",
		PdeathSignal:  0,
		Setpgid:       false,
		Criu:          "",
		SystemdCgroup: false,
		Rootless:      nil,
	}

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	bundlePath := tmpDir.Path()
	cliLog := logger.NewCliLogger(constant.CliLogBufferLen)
	spec := &specs.Spec{}

	ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
	runner := NewOCIRunner(&OCIRunOpts{
		Ctx:         ctx,
		Spec:        spec,
		RuntimePath: "",
		BundlePath:  bundlePath,
		NoPivot:     false,
		Output:      cliLog,
	})
	runner.runtime = runtime
	_, err := runner.runContainer()
	assert.ErrorContains(t, err, "context finished")
}

func TestRunOCIRuntimeFailed(t *testing.T) {
	runtime := &mockFailRunc{
		Command:       "",
		Root:          "",
		Debug:         false,
		Log:           "",
		LogFormat:     "",
		PdeathSignal:  0,
		Setpgid:       false,
		Criu:          "",
		SystemdCgroup: false,
		Rootless:      nil,
	}

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	bundlePath := tmpDir.Path()
	cliLog := logger.NewCliLogger(constant.CliLogBufferLen)
	spec := &specs.Spec{}

	runner := NewOCIRunner(&OCIRunOpts{
		Ctx:         context.Background(),
		Spec:        spec,
		RuntimePath: "",
		BundlePath:  bundlePath,
		NoPivot:     false,
		Output:      cliLog,
	})
	runner.runtime = runtime
	_, err := runner.runContainer()
	assert.ErrorContains(t, err, "run error")
}

func TestReadPidFail(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	bundlePath := tmpDir.Path()

	_, err := readPid(bundlePath + "/pid")
	assert.ErrorContains(t, err, "no such file or directory")

	err = ioutil.WriteFile(bundlePath+"/pid", []byte("fakepid"), constant.DefaultSharedFileMode)
	assert.NilError(t, err)

	_, err = readPid(bundlePath + "/pid")
	assert.ErrorContains(t, err, "parsing pid fakepid err")
}

func TestWaitPidFail(t *testing.T) {
	wstatus, err := waitPid(999)
	assert.Equal(t, wstatus.ExitStatus(), 0)
	assert.ErrorContains(t, err, "no child processes")
}
