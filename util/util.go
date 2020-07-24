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

package util

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/containers/storage/pkg/fileutils"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
	"github.com/docker/docker/pkg/signal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	constant "isula.org/isula-build"
)

// LogFieldKey used to specifying a field for logrus
type LogFieldKey string

// BuildDirKey is type used for BuildDir in build context
type BuildDirKey string

const (
	// DefaultTransport is default transport
	DefaultTransport = "docker://"

	// LogKeyBuildID describes the key field with buildID for logrus
	LogKeyBuildID = "buildID"

	// BuildDir describes the key field with BuildDir in build context
	BuildDir = "buildDir"
)

var (
	// DefaultRegistryPathPrefix is the map for registry and path
	DefaultRegistryPathPrefix map[string]string
	// clientExporters to map exporter whether will export the image to client
	clientExporters map[string]bool
)

func init() {
	DefaultRegistryPathPrefix = map[string]string{
		"index.docker.io": "library",
		"docker.io":       "library",
	}
	clientExporters = map[string]bool{
		"docker-archive": true,
		"oci-archive":    true,
		"isulad":         true,
	}
}

// IsClientExporter used to determinate exporter whether need to send the image to client
func IsClientExporter(exporter string) bool {
	_, ok := clientExporters[exporter]
	return ok
}

// GetIgnorePatternMatcher returns docker ignore matcher
func GetIgnorePatternMatcher(ignores []string, dir, excludeDir string) (*fileutils.PatternMatcher, error) {
	patterns := []string{excludeDir}
	for _, ignore := range ignores {
		prefix := ""
		if ignore[0] == '!' {
			prefix = string(ignore[0])
			ignore = ignore[1:]
		}
		if ignore == "" {
			continue
		}
		patterns = append(patterns, prefix+filepath.Join(dir, ignore))
	}

	return fileutils.NewPatternMatcher(patterns)
}

// IsMatched returns true when it matches the path
func IsMatched(matcher *fileutils.PatternMatcher, path string) (bool, error) {
	result, err := matcher.MatchesResult(path)
	if err != nil {
		return false, err
	}

	return result.IsMatched(), nil
}

// CopyURLResource gets file from url and copies it into dest
func CopyURLResource(ctx context.Context, url, dest string, uid, gid int) (err error) {
	c := &http.Client{
		Timeout: constant.DefaultHTTPTimeout,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to new a request %q", url)
	}

	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error getting %q", url)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logrus.Warningf("Closing resp.Body failed: %v", cerr)
		}
	}()

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		err2 := f.Close()
		if err == nil {
			err = err2
		}
	}()

	if err = f.Chmod(constant.DefaultRootFileMode); err != nil {
		return err
	}

	if err = f.Chown(uid, gid); err != nil {
		return err
	}

	logrus.Debugf("Get file from url %q and copies it into dest %q", url, dest)
	w := bufio.NewWriter(f)
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return errors.Errorf("failed to correctly read from %q, the length wanted %q, "+
			"the actual length %q", url, resp.ContentLength, n)
	}

	return w.Flush()
}

// CopySymbolFile copies symbol file
func CopySymbolFile(src, dest string, chownPair idtools.IDPair) error {
	// remove dest regular file
	if destFi, err := os.Lstat(dest); err == nil {
		if destFi.IsDir() {
			return nil
		}
		if err2 := os.Remove(dest); err2 != nil {
			return err2
		}
	}

	target, err := os.Readlink(src)
	if err != nil {
		return err
	}

	if err = os.Symlink(target, dest); err != nil {
		return err
	}

	return os.Lchown(dest, chownPair.UID, chownPair.GID)
}

// CopyFile copies a single file
func CopyFile(src, dest string, chownPair idtools.IDPair) error {
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcFileInfo.Mode().IsRegular() {
		return nil
	}

	src, dest = filepath.Clean(src), filepath.Clean(dest)
	srcFile, err := os.Open(src) // nolint:gosec
	if err != nil {
		return err
	}

	defer func() {
		if cerr := srcFile.Close(); cerr != nil {
			logrus.Warningf("Closing %q failed: %v", src, cerr)
		}
	}()

	if err = idtools.MkdirAllAndChownNew(filepath.Dir(dest), constant.DefaultSharedDirMode, chownPair); err != nil {
		return errors.Wrapf(err, "error creating directory %q", filepath.Dir(dest))
	}
	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil {
			logrus.Warningf("Closing %q failed: %v", dest, cerr)
		}
	}()

	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	if n != srcFileInfo.Size() {
		return errors.Errorf("copied length %v, expected %v", n, srcFileInfo.Size())
	}

	if err = os.Chown(dest, chownPair.UID, chownPair.GID); err != nil {
		return err
	}

	if err = os.Chmod(dest, srcFileInfo.Mode()); err != nil {
		return err
	}

	return CopyXattrs(src, dest)
}

// ValidateSignal check and returns the signal of type syscall.Signal
func ValidateSignal(sigStr string) (syscall.Signal, error) {
	const minSignalInt, maxSignalInt = 0, 64
	// check if sigStr is integer type(like 9) or string type(like SIGKILL)
	if s, err := strconv.Atoi(sigStr); err == nil {
		if s <= minSignalInt || s > maxSignalInt {
			return -1, errors.Errorf("invalid signal: %s", sigStr)
		}
		return syscall.Signal(s), nil
	}
	sig, ok := signal.SignalMap[strings.TrimPrefix(strings.ToUpper(sigStr), "SIG")]
	if !ok {
		return -1, errors.Errorf("invalid signal: %s", sigStr)
	}
	return sig, nil
}

// GenerateCryptoNum generates secure random number with length in range 1-19
// The output type is string instead of int
func GenerateCryptoNum(s int) (string, error) {
	if s > 19 || s < 1 {
		return "", errors.Errorf("generate random number failed with input range %d, should be in range 1-19", s)
	}
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return "", err
	}
	res := fmt.Sprintf("%d", num)
	return res[:s], nil
}

func copyXattrByKey(src, dest, key string) error {
	attrValue, err := system.Lgetxattr(src, key)
	if err != nil && err != unix.EOPNOTSUPP {
		return err
	}

	if attrValue == nil {
		return nil
	}

	return system.Lsetxattr(dest, key, attrValue, 0)
}

// CopyXattrs copies xattrs from src to dest file
func CopyXattrs(src, dest string) error {
	xattrs, err := system.Llistxattr(src)
	if err != nil && err != unix.EOPNOTSUPP {
		return err
	}

	for _, key := range xattrs {
		if strings.HasPrefix(key, "security.") || strings.HasPrefix(key, "user.") {
			if err := copyXattrByKey(src, dest, key); err != nil {
				return err
			}
		}
	}

	return nil
}
