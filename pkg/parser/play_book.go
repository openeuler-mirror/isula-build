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
// Description: playbook related functions

package parser

// A PlayBook is the parse result of the Dockerfile. It construct with
// one or more pages which stand for the stage of an image build
type PlayBook struct {
	Pages       []*Page
	HeadingArgs []string
	Warnings    []string
}

// Dump the play book with string
func (p *PlayBook) Dump() string {
	if p == nil {
		return ""
	}
	str := ""
	for _, page := range p.Pages {
		str += page.dump()
	}

	return str
}
