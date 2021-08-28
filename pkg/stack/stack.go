// +build !windows
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
// Description: provides stack dump related functions

// Package stack includes stack print related functions
package stack

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	constant "isula.org/isula-build"
)

// Setup brings up the signal handler to dump runtime stack
func Setup(path string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, unix.SIGUSR1)
	go func() {
		for range sigCh {
			if err := logRotate(path); err == nil {
				dumpStack(path)
			}
		}
	}()
}

const (
	// remember to keep stackLogFormat and stackLogRegStr with same format
	stackLogFormat = "stack-%s.log"
	stackLogRegStr = `^stack-[0-9T\-\+]{22}.log$`
	maxStackLogs   = 5
	// stackBufSize refer to _MaxSmallSize in golang/src/runtime/sizeclasses.go
	stackBufSize = 32768
	// stackBufFactor is the factor of stack increase speed
	stackBufFactor = 2
)

func dumpStack(path string) {
	var (
		stackBuf  []byte
		stackSize int
		bufSize   = stackBufSize
	)

	for {
		stackBuf = make([]byte, bufSize)
		stackSize = runtime.Stack(stackBuf, true)
		// if these two sizes equal, which means the allocated buf is not large enough to carry all
		// stacks back, so enlarge the buf and try again
		if stackSize != bufSize {
			break
		}
		bufSize *= stackBufFactor
	}

	fp := filepath.Join(path, fmt.Sprintf(stackLogFormat, time.Now().Format("2006-01-02T150405+0800")))
	if err := ioutil.WriteFile(fp, stackBuf[:stackSize], constant.DefaultRootFileMode); err != nil {
		logrus.Errorf("Writing runtime stack to file %q failed: %v", fp, err)
		return
	}
	logrus.Infof("Written runtime stack to %s", fp)
}

// logRotate rotates stack logs with maxStackLogs, avoiding too much stack logs booming memory or disk
func logRotate(path string) error {
	var (
		regStackLog = regexp.MustCompile(stackLogRegStr)
		files       sort.StringSlice
	)

	items, err := ioutil.ReadDir(path)
	if err != nil {
		logrus.Warningf("Read dir %s failed: %v", path, err)
		return err
	}

	for _, fi := range items {
		if regStackLog.MatchString(fi.Name()) {
			files = append(files, fi.Name())
		}
	}

	// try best to rotate old logs, only returns last err to caller
	var retErr error
	sort.Sort(sort.Reverse(files[:]))
	for i := maxStackLogs - 1; i < len(files); i++ {
		logrus.Infof("Stack log rotating: %s", files[i])
		if err := os.Remove(filepath.Join(path, files[i])); err != nil {
			logrus.Warningf("Remove %s failed: %v", files[i], err)
			retErr = err
		}
	}
	return retErr
}
