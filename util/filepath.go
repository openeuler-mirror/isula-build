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
// Description: filepath related common functions

package util

import (
	"os"
	"path/filepath"
	"strings"
)

// HasSlash is used to determine whether the input path ends with a slash (/)
func HasSlash(path string) bool {
	return len(path) > 0 && (strings.HasSuffix(path, string(os.PathSeparator)) || strings.HasSuffix(path, string(os.PathSeparator)+"."))
}

// MakeAbsolute make the provided path absolutely
func MakeAbsolute(path, workingDir string) string {
	// make the path relative to the current WORKINGDIR
	if path == "." {
		if !HasSlash(workingDir) {
			workingDir += string(os.PathSeparator)
		}
		path = workingDir
	}

	if !filepath.IsAbs(path) {
		hasSlash := HasSlash(path)
		path = filepath.Join(string(os.PathSeparator), filepath.FromSlash(workingDir), path)

		if hasSlash {
			path += string(os.PathSeparator)
		}
	}

	return path
}

// IsDirectory returns true if the file exists and it is a dir
func IsDirectory(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

// IsExist returns true if the path exists
func IsExist(path string) bool {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// IsSymbolFile returns true if the path file is a symbol file
func IsSymbolFile(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}

	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		return true
	}

	return false
}
