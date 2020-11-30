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
// Create: 2020-04-01
// Description: common functions

// Package util includes common used functions
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	constant "isula.org/isula-build"
)

const (
	maxServerNameLength = 255
	maxLoadFileSize     = 50 * 1024 * 1024 * 1024
)

// CopyMapStringString copies all KVs in a map[string]string to a new map
func CopyMapStringString(m map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result[k] = v
	}
	return result
}

// CopyStrings copies all strings in a slice to a new slice
func CopyStrings(str []string) []string {
	result := make([]string, len(str), len(str))
	copy(result, str)
	return result
}

// CopyStringsWithoutSpecificElem copies the string without specified substring in a slice to a new slice
func CopyStringsWithoutSpecificElem(str []string, e string) []string {
	result := make([]string, 0, len(str))
	for _, s := range str {
		if !strings.Contains(s, e) {
			result = append(result, s)
		}
	}

	return result
}

// NoArgs is used for command which has no args
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return errors.Errorf("%q accepts no arguments.\nSee '%s --help'. \n\nExample:   %s",
		cmd.CommandPath(),
		cmd.CommandPath(),
		cmd.Example,
	)
}

// FlagErrorFunc is used to print error message when invalid flag input
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	usage := ""
	if cmd.HasSubCommands() {
		usage = "\n\n" + cmd.UsageString()
	}
	return errors.Errorf("%s\nSee '%s --help'.%s", err, cmd.CommandPath(), usage)
}

// SetUmask try setting the umask for current process to DefaultUmask
func SetUmask() bool {
	wanted := constant.DefaultUmask
	unix.Umask(wanted)
	return unix.Umask(wanted) == wanted
}

// CheckFileSize check whether the file size exceeds limit
func CheckFileSize(path string, sizeLimit int64) error {
	filename := filepath.Base(path)
	f, err := os.Stat(filepath.Clean(path))
	// file not exist, file size check ok
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Errorf("stat file %v err: %v", filename, err)
	}
	if f.IsDir() {
		return errors.Errorf("file %s is a directory", filename)
	}
	if f.Size() > sizeLimit {
		return errors.Errorf("file %v size is: %v, exceeds limit %v", filename, f.Size(), sizeLimit)
	}

	return nil
}

// CheckLoadFile checks the file which will be loaded
func CheckLoadFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return errors.Wrapf(err, "stat %q failed", path)
	}

	if !fi.Mode().IsRegular() {
		return errors.Errorf("loading file %s should be a regular file", fi.Name())
	}

	if fi.Size() == 0 {
		return errors.New("loading file is empty")
	}

	if fi.Size() > maxLoadFileSize {
		return errors.Errorf("file %s size is: %v, exceeds limit %v", fi.Name(), fi.Size(), maxLoadFileSize)
	}

	return nil
}

// ParseServer will get registry address from input
// if input is https://index.docker.io/v1
// the result will be index.docker.io
func ParseServer(server string) (string, error) {
	if len(server) > maxServerNameLength {
		return "", errors.New("max length of server name exceeded")
	}
	// first trim prefix https:// and http://
	server = strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://")
	// then trim prefix docker://
	server = strings.TrimPrefix(server, DefaultTransport)
	// always get first part split by "/"
	fields := strings.Split(server, "/")
	if fields[0] == "" {
		return "", errors.Errorf("invalid registry address %s", server)
	}

	// to prevent directory traversal
	fakePrefix := "/fakePrefix"
	origAddr := fmt.Sprintf("%s/%s", fakePrefix, fields[0])
	cleanAddr, err := securejoin.SecureJoin(fakePrefix, fields[0])
	if err != nil {
		return "", err
	}
	if cleanAddr != origAddr {
		return "", errors.Errorf("invalid relative path detected")
	}

	return fields[0], nil
}

// FormatSize formats size using powers of base(1000 or 1024)
func FormatSize(size, base float64) string {
	suffixes := [5]string{"B", "KB", "MB", "GB", "TB"}
	cnt := 0
	for size >= base && cnt < len(suffixes)-1 {
		size /= base
		cnt++
	}

	return fmt.Sprintf("%.3g %s", size, suffixes[cnt])
}
