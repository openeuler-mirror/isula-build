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
// Create: 2020-1-20
// Description: This file contains utils used for test cases

// Package util contains utils used for test cases
package util

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os/exec"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

const (
	// TestRegistryKey describes the registry used in all of the test case
	TestRegistryKey string = "TEST_REG"
	// SkipRegTestKey describes if it is defined, skip the current test case
	SkipRegTestKey string = "SKIP_REG"
)

var (
	// DefaultTestRegistry is default registry used for test case
	DefaultTestRegistry = "docker.io"
)

// GetTestingArgs receives flag passed by test
func GetTestingArgs(t *testing.T) map[string]string {
	args := make(map[string]string)
	if !flag.Parsed() {
		flag.Parse()
	}

	for _, arg := range flag.Args() {
		items := strings.Split(arg, "=")
		if len(items) == 2 {
			args[items[0]] = items[1]
		} else if len(items) == 1 {
			args[items[0]] = ""
		} else {
			fmt.Println("WARN: invalid testing args:", arg)
		}
	}

	return args
}

// Immutable is used to set immutable
func Immutable(path string, set bool) error {
	var op string
	if set {
		op = "+i" // set immutable
	} else {
		op = "-i" // set mutable
	}
	cmd := exec.Command("chattr", op, path) // nolint:gosec
	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "chattr %s for %s failed", op, path)
	}
	return nil
}

// GenRandInt64 is to generate an nondeterministic int64 value
func GenRandInt64() int64 {
	val, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return val.Int64()
}
