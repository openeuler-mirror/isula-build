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
// Create: 2020-07-22
// Description: This file is used for opts test

package opts

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestOptValidator(t *testing.T) {
	type testcase struct {
		name   string
		value  string
		expect [2]string
		isErr  bool
		errStr string
	}
	var testcases = []testcase{
		{
			name:   "valid",
			value:  "k=v",
			expect: [2]string{"k", "v"},
		},
		{
			name:   "invalid",
			value:  "k,v",
			isErr:  true,
			errStr: "invalid format",
		},
	}

	for _, tc := range testcases {
		k, v, err := OptValidator(tc.value)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.Equal(t, tc.expect[0], k)
			assert.Equal(t, tc.expect[1], v)
		}
	}

}

func TestStringa(t *testing.T) {
	type testcase struct {
		name   string
		value  string
		isErr  bool
		errStr string
	}
	var testcases = []testcase{
		{
			name:  "valid",
			value: "k=v",
		},
		{
			name:   "invalid",
			value:  "k,v",
			isErr:  true,
			errStr: "invalid format",
		},
	}

	for _, tc := range testcases {
		listOpt := NewListOpts(OptValidator)
		err := listOpt.Set(tc.value)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.ErrorContains(t, err, tc.errStr, tc.name)
		}
		if err == nil {
			str := listOpt.String()
			assert.Equal(t, str, tc.value, tc.name)
		}
	}

}
