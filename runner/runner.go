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
// Description: runner related functions

// Package runner is used to execute RUN command
package runner

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/go-runc"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	constant "isula.org/isula-build"
	"isula.org/isula-build/util"
)

const (
	defaultRuntime = "runc"
)

// Runner is the executor to execute RUN command
type Runner interface {
	Run() error
}

// OCIRuntime is runtime used to running container according to OCI standardization
type OCIRuntime interface {
	Run(context context.Context, id, bundle string, opts *runc.CreateOpts) (int, error)
	Kill(context context.Context, id string, sig int, opts *runc.KillOpts) error
	Delete(context context.Context, id string, opts *runc.DeleteOpts) error
	State(context context.Context, id string) (*runc.Container, error)
}

// OCIRunner is Runner running container according to OCI standardization
type OCIRunner struct {
	spec        *specs.Spec
	ctx         context.Context
	runtimePath string
	bundlePath  string
	noPivot     bool
	output      io.Writer
	runtime     OCIRuntime
}

// OCIRunOpts is OCIRunner options
type OCIRunOpts struct {
	Spec        *specs.Spec
	Ctx         context.Context
	RuntimePath string
	NoPivot     bool
	BundlePath  string
	Output      io.Writer
}

// NewOCIRunner creates a new OCIRunner
func NewOCIRunner(opts *OCIRunOpts) *OCIRunner {
	if opts.RuntimePath == "" {
		opts.RuntimePath = defaultRuntime
	}
	runtime := &runc.Runc{
		Command:      opts.RuntimePath,
		LogFormat:    runc.JSON,
		PdeathSignal: syscall.SIGKILL,
		Setpgid:      true,
	}

	return &OCIRunner{
		ctx:         opts.Ctx,
		spec:        opts.Spec,
		runtimePath: opts.RuntimePath,
		bundlePath:  opts.BundlePath,
		noPivot:     opts.NoPivot,
		output:      opts.Output,
		runtime:     runtime,
	}
}

// Run runs a container to execute specified command
func (r *OCIRunner) Run() error {
	// write spec to file config.json
	specBytes, err := json.Marshal(r.spec)
	if err != nil {
		return errors.Errorf("encoding configuration as json err: %v", err)
	}
	if err = ioutils.AtomicWriteFile(filepath.Join(r.bundlePath, "config.json"), specBytes, constant.DefaultRootFileMode); err != nil {
		return errors.Errorf("write spec to config.json err: %v", err)
	}

	status, err := r.runContainer()
	if err != nil {
		return errors.Errorf("runOCIRuntime err: %v", err)
	}

	if status.Exited() && status.ExitStatus() != 0 {
		return errors.Errorf("container exited error with status: %v", status.ExitStatus())
	}
	if status.Signaled() {
		return errors.Errorf("container exited on: %v", status.Signal())
	}

	return nil
}

func (r *OCIRunner) runContainer() (unix.WaitStatus, error) { // nolint:gocyclo
	var (
		pid     int
		wstatus unix.WaitStatus
	)

	pLog := logrus.WithField(util.LogKeySessionID, r.ctx.Value(util.LogFieldKey(util.LogKeySessionID)))
	containerName := filepath.Base(r.bundlePath)
	pidFile := filepath.Join(r.bundlePath, "pid")
	createOpts := runc.CreateOpts{
		IO:      &forwardIO{stdin: nil, stdout: r.output, stderr: r.output},
		PidFile: pidFile,
		Detach:  true,
		NoPivot: r.noPivot,
	}

	defer func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stateOutput, stateErr := r.runtime.State(ctx, containerName)
		if stateErr == nil && stateOutput.Status != "stopped" {
			killErr := r.runtime.Kill(ctx, containerName, int(syscall.SIGKILL), nil)
			if killErr != nil {
				if pid > 1 {
					killErr = unix.Kill(pid, syscall.SIGKILL)
				}
				pLog.Warnf("Kill container %v error: %v", containerName, killErr)
			}
		}

		if deleteErr := r.runtime.Delete(ctx, containerName, nil); deleteErr != nil {
			pLog.Warnf("Delete container %v error: %v,", containerName, deleteErr)
		}
	}()

	eg, _ := errgroup.WithContext(r.ctx)
	eg.Go(func() error {
		pLog.Debugf("Running container: %v", containerName)
		if _, err := r.runtime.Run(r.ctx, containerName, r.bundlePath, &createOpts); err != nil {
			return errors.Wrap(err, "error running container")
		}

		return nil
	})

	// read the container's exit status when it exits.
	eg.Go(func() error {
		var rErr, wErr error
		pidGetWaitDuration, pidGetWaitTimes := 100*time.Millisecond, 100
		for i := 0; i < pidGetWaitTimes; i++ {
			if pid, rErr = readPid(pidFile); rErr == nil {
				break
			}
			time.Sleep(pidGetWaitDuration)
		}
		if rErr != nil {
			return errors.New("timeout to get container init process pid")
		}

		if wstatus, wErr = waitPid(pid); wErr != nil {
			pLog.Errorf("Error waiting for process %d, wstatus: %v, err: %v", pid, wstatus, wErr)
		}

		return nil
	})

	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		errC <- eg.Wait()
	}()

	select {
	case <-r.ctx.Done():
		return 1, errors.Wrap(r.ctx.Err(), "context finished")
	case err, ok := <-errC:
		if !ok {
			pLog.Info("Channel errC closed")
			return 1, nil
		}
		if err != nil {
			return 1, err
		}
	}

	return wstatus, nil
}

func readPid(pidFilePath string) (int, error) {
	if err := util.CheckFileInfoAndSize(pidFilePath, constant.MaxFileSize); err != nil {
		return 0, err
	}
	pidValue, err := ioutil.ReadFile(filepath.Clean(pidFilePath))
	if err != nil {
		return 0, errors.Errorf("reading pid from %v err: %v", pidFilePath, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidValue)))
	if err != nil {
		return 0, errors.Errorf("parsing pid %s err: %v", string(pidValue), err)
	}

	return pid, nil
}

func waitPid(pid int) (unix.WaitStatus, error) {
	var (
		wstatus unix.WaitStatus
		err     error
	)
	// refer to golang.org/x/sys/unix/syscall_linux.go:325, exited is 0x00
	const exitedStatus = 0x00
	if _, err = unix.Wait4(pid, &wstatus, 0, nil); err != nil {
		wstatus = exitedStatus
	}

	return wstatus, err
}

type forwardIO struct {
	stdin          io.ReadCloser
	stdout, stderr io.Writer
}

func (f *forwardIO) Close() error {
	return nil
}

func (f *forwardIO) Stdin() io.WriteCloser {
	return nil
}

func (f *forwardIO) Stdout() io.ReadCloser {
	return nil
}

func (f *forwardIO) Stderr() io.ReadCloser {
	return nil
}

func (f *forwardIO) Set(cmd *exec.Cmd) {
	cmd.Stdin = f.stdin
	cmd.Stdout = f.stdout
	cmd.Stderr = f.stderr
}
