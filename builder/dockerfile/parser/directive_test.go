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
// Create: 2020-03-20
// Description: directive related functions tests

package dockerfile

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestInitDirective(t *testing.T) {
	type testcase struct {
		name   string
		expect string
		err    string
	}
	var testcases = []testcase{
		{
			name:   "directive_with_comment_head",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_with_no_space_line",
			expect: "`",
			err:    "",
		},
		{
			name:   "directive_with_space_line",
			expect: "`",
			err:    "",
		},
		{
			name:   "directive_with_space",
			expect: "`",
			err:    "",
		},
		{
			name:   "directive_with_double_quote",
			expect: "`",
			err:    "",
		},
		{
			name:   "slash_directive_with_no_space_line",
			expect: "\\",
			err:    "\\",
		},
		{
			name:   "slash_directive_with_space_line",
			expect: "\\",
			err:    "",
		},
		{
			name:   "slash_directive_with_space_line",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_with_double_slash",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_with_continues_line",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_after_build_cmd",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_after_comment",
			expect: "\\",
			err:    "",
		},
		{
			name:   "unknow_directive",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_after_unkonw_directive",
			expect: "\\",
			err:    "",
		},
		{
			name:   "directive_with_double",
			expect: "\\",
			err:    "only support one escape directive",
		},
		{
			name:   "directive_with_illegal_char",
			expect: "\\",
			err:    "invalid escape token",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "directive", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()
			d, err := newDirective(r)
			if err != nil {
				assert.ErrorContains(t, err, tc.err)
			} else {
				assert.Equal(t, string(d.escapeToken), tc.expect)
			}
		})
	}
}
