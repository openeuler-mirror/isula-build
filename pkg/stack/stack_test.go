// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jingxiao Lu
// Create: 2020-03-20
// Description: stack print related functions tests

package stack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/containers/storage/pkg/reexec"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gotest.tools/assert"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	testutil "isula.org/isula-build/tests/util"
)

var (
	testDumpStackCommand = "TestDumpStack"
	pidfile              = "pidfile"
)

func init() {
	reexec.Register(testDumpStackCommand, testDumpStack)
}

func testDumpStack() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Printf("testDumpStack get pwd failed: %v\n", err)
		os.Exit(constant.DefaultFailedCode)
	}

	Setup(root)
	if err = ioutil.WriteFile(path.Join(root, pidfile), []byte(strconv.Itoa(os.Getpid())), constant.DefaultSharedDirMode); err != nil {
		fmt.Printf("testDumpStack write pidfile failed: %v\n", err)
		os.Exit(constant.DefaultFailedCode)
	}
	time.Sleep(30 * time.Second)
}

func TestSetup(t *testing.T) {
	if reexec.Init() {
		return
	}

	var testDir *fs.Dir
	testDir = fs.NewDir(t, "TestSetup")
	defer testDir.Remove()

	cmd := reexec.Command(testDumpStackCommand)
	cmd.Dir = testDir.Path()

	var eg errgroup.Group
	eg.Go(func() error {
		// start a separate process to handle SIGUSR1 and dump stacks
		err := cmd.Run()
		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		pidfilePath := path.Join(testDir.Path(), pidfile)

		// wait for the pidfile created
		now := time.Now()
		timeout := time.After(10 * time.Second)
		select {
		case <-timeout:
			return errors.New("TestSetup wait for TestDump timeout")
		case <-time.After(time.Until(now.Add(100 * time.Millisecond))):
			if _, err := os.Stat(pidfilePath); os.IsExist(err) {
				break
			}
		}

		pidValue, err := ioutil.ReadFile(pidfilePath)
		if err != nil {
			return errors.Wrapf(err, "TestSetup read pidfile failed")
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidValue)))
		if err != nil {
			return errors.Wrapf(err, "TestSetup convert pid string %s to int failed", pidValue)
		}
		if err = syscall.Kill(pid, syscall.SIGUSR1); err != nil {
			return errors.Wrapf(err, "TestSetup sending SIGUSR1 to %d failed", pid)
		}

		time.Sleep(1 * time.Second)
		var found = false
		err = filepath.Walk(cmd.Dir, func(path string, info os.FileInfo, err error) error {
			// the stack file is "stack-%s.log"
			if strings.Contains(path, "stack") {
				found = true
			}
			return nil
		})
		assert.NilError(t, err)
		assert.Equal(t, found, true)

		// testing passed, kill it
		return syscall.Kill(pid, syscall.SIGKILL)
	})

	err := eg.Wait()
	assert.NilError(t, err)
}

func TestReqStackLog(t *testing.T) {
	var tests = []struct {
		name     string
		filename string
		match    bool
	}{
		{
			name:     "test 1",
			filename: "stack-2020-04-08T213927+0800.log",
			match:    true,
		},
		{
			name:     "test 2",
			filename: "stack-2020-04-08T21327+0800.log",
			match:    false,
		},
		{
			name:     "test 3",
			filename: "stack-2020-04-08T2221327+0800.log",
			match:    false,
		},
		{
			name:     "test 4",
			filename: "stack-2020-04-08T221327+0800.logstack-2020-04-08T221327+0800.log",
			match:    false,
		},
	}

	var regStackLog = regexp.MustCompile(stackLogRegStr)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if regStackLog.MatchString(tt.filename) != tt.match {
				t.FailNow()
			}
		})
	}
}

func TestDumpStack(t *testing.T) {
	dir := fs.NewDir(t, "TestDumpStack")
	defer dir.Remove()

	var regStackLog = regexp.MustCompile(stackLogRegStr)
	dumpStack(dir.Path())

	var found bool
	items, err := ioutil.ReadDir(dir.Path())
	assert.NilError(t, err)
	for _, fi := range items {
		if regStackLog.MatchString(fi.Name()) {
			found = true
		}
	}
	assert.Equal(t, found, true)
}

// TestDumpStackWriteFail dumpStack fail at WriteFile to the path
func TestDumpStackWriteFail(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/var/tmp", t.Name())
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer os.RemoveAll(tmpDir)
	if err = testutil.Immutable(tmpDir, true); err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer testutil.Immutable(tmpDir, false)

	var regStackLog = regexp.MustCompile(stackLogRegStr)
	dumpStack(tmpDir)

	var found bool
	items, err := ioutil.ReadDir(tmpDir)
	assert.NilError(t, err)
	for _, fi := range items {
		if regStackLog.MatchString(fi.Name()) {
			found = true
		}
	}
	assert.Equal(t, found, false)
}

func TestLogRotate(t *testing.T) {
	dir := fs.NewDir(t, "TestLogRotate")
	defer dir.Remove()

	for i := 0; i < maxStackLogs+2; i++ {
		fp := filepath.Join(dir.Path(), fmt.Sprintf(stackLogFormat, time.Now().Format("2006-01-02T150405+0800")))
		err := ioutil.WriteFile(fp, []byte("goroutines"), constant.DefaultRootFileMode)
		assert.NilError(t, err)
		time.Sleep(1 * time.Second)
	}
	items, err := ioutil.ReadDir(dir.Path())
	assert.NilError(t, err)
	assert.Equal(t, len(items), maxStackLogs+2)

	logRotate(dir.Path())
	items, err = ioutil.ReadDir(dir.Path())
	assert.NilError(t, err)
	// logRotate will keep only "maxStackLogs-1" items. Then will the new dumped stack, we have maxStackLogs items
	assert.Equal(t, len(items), maxStackLogs-1)
}

func TestLogRotateWrongPath(t *testing.T) {
	dirPath := "/abc/foo"
	if err := logRotate(dirPath); err == nil {
		t.FailNow()
	}
}

// TestLogRotateFail fails for rotating old logs for removing
func TestLogRotateFail(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/var/tmp", t.Name())
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < maxStackLogs+2; i++ {
		fp := filepath.Join(tmpDir, fmt.Sprintf(stackLogFormat, time.Now().Format("2006-01-02T150405+0800")))
		err = ioutil.WriteFile(fp, []byte("goroutines"), constant.DefaultRootFileMode)
		assert.NilError(t, err)
		time.Sleep(1 * time.Second)
	}
	items, err := ioutil.ReadDir(tmpDir)
	assert.NilError(t, err)
	assert.Equal(t, len(items), maxStackLogs+2)

	if err = testutil.Immutable(tmpDir, true); err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer testutil.Immutable(tmpDir, false)

	if err = logRotate(tmpDir); err == nil {
		t.Log(err)
		t.FailNow()
	}
	items, err = ioutil.ReadDir(tmpDir)
	assert.NilError(t, err)
	// logRotate failed for immutable dir, so the items are still maxStackLogs+2
	assert.Equal(t, len(items), maxStackLogs+2)
}
