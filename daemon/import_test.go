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
// Create: 2020-07-25
// Description: This is test file for daemon import.go

package daemon

import (
	"testing"

	"gotest.tools/assert"
)

func TestParseReference(t *testing.T) {
	type testcase struct {
		name      string
		reference string
		expect    string
		isErr     bool
		errStr    string
	}
	var testcases = []testcase{
		{
			name:      "repo only",
			reference: "busybox",
			expect:    "busybox",
		},
		{
			name:      "repo and tag",
			reference: "busybox:latest",
			expect:    "busybox:latest",
		},
		{
			name:      "ref with tag missing",
			reference: "busybox:",
			isErr:     true,
			errStr:    "invalid reference format",
		},
		{
			name:      "empty ref",
			reference: "",
			expect:    noneReference,
		},
		{
			name:      "ref with no tag",
			reference: "busybox",
			expect:    "busybox",
		},
		{
			name:      "ref with space",
			reference: "busybox: latest",
			isErr:     true,
			errStr:    "invalid reference format",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := parseReference(tc.reference)
			assert.Equal(t, err != nil, tc.isErr, tc.name)
			if err != nil {
				assert.ErrorContains(t, err, tc.errStr, tc.name)
			}
			if err == nil {
				assert.Equal(t, ref, tc.expect, tc.name)
			}
		})
	}
}
