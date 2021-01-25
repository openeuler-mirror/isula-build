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
// Description: This file is used for playbook test

package parser

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDump(t *testing.T) {
	page := &Page{
		Name:  "test",
		Begin: 1,
		End:   2,
	}

	cell10 := &Cell{"busybox:latest"}
	cell11 := &Cell{"as"}
	cell12 := &Cell{"mybusybox"}

	cell20 := &Cell{"pwd"}

	line1 := &Line{
		Begin:   1,
		End:     1,
		Command: "FROM",
	}
	line1.AddCell(cell10)
	line1.AddCell(cell11)
	line1.AddCell(cell12)
	ret := line1.IsJSONArgs()
	assert.Assert(t, !ret)

	line2 := &Line{
		Begin:   2,
		End:     2,
		Command: "RUN",
		Flags:   map[string]string{"attribute": "json"},
	}
	line2.AddCell(cell20)
	ret = line2.IsJSONArgs()
	assert.Assert(t, ret)

	page.AddLine(line1)
	page.AddLine(line2)

	p := &PlayBook{
		Pages: []*Page{page},
	}

	str := p.Dump()
	assert.Equal(t, str, "  1-1  | (FROM) (busybox:latest) (as) (mybusybox) \n  2-2  | (RUN) (pwd) ")
}
