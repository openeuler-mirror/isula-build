// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu
// Create: 2020-05-18
// Description: user related common functions tests

package util

import (
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestCheckFileSize(t *testing.T) {
	content := `
 ARG testArg
 ARG testArg2
 ARG testArg3=val
 ARG testArg=val
 FROM alpine AS uuid
 RUN touch abc
 `

	type testCase struct {
		name      string
		sizeLimit int64
		ctxDir    *fs.Dir
		isDir     bool
		isErr     bool
		errStr    string
	}
	cases := []testCase{
		{
			name:      "leagal file",
			ctxDir:    fs.NewDir(t, t.Name(), fs.WithFile("Dockerfile", content)),
			sizeLimit: 200,
		},
		{
			name:      "exceeds limit file",
			ctxDir:    fs.NewDir(t, t.Name(), fs.WithFile("Dockerfile", content)),
			sizeLimit: 50,
			isErr:     true,
			errStr:    "exceeds limit",
		},
		{
			name:   "directory",
			ctxDir: fs.NewDir(t, t.Name(), fs.WithFile("Dockerfile", content)),
			isDir:  true,
			isErr:  true,
			errStr: "is a directory",
		},
		{
			name:      "not exist file",
			ctxDir:    fs.NewDir(t, t.Name()),
			sizeLimit: 5,
		},
		{
			name:      "exceeds limit directory",
			ctxDir:    fs.NewDir(t, t.Name(), fs.WithFile("Dockerfile", content)),
			sizeLimit: 50,
			isErr:     true,
			errStr:    "exceeds limit",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer c.ctxDir.Remove()
			path := c.ctxDir.Path()
			if !c.isDir {
				path = filepath.Join(path, "Dockerfile")
			}
			err := CheckFileSize(path, c.sizeLimit)
			assert.Equal(t, err != nil, c.isErr)
			if c.isErr {
				assert.ErrorContains(t, err, c.errStr)
			}
		})
	}
}
