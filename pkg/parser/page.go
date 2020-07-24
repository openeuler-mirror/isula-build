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
// Description: page related functions

package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// Page is a Dockerfile stage
type Page struct {
	Lines []*Line // all lines which the page contains
	Name  string  // page name
	Begin int     // the begin line number of the page in the physical Dockerfile
	End   int     // the end line number of the page in the physical Dockerfile
}

func (p *Page) dump() string {
	var fields []string
	for _, line := range p.Lines {
		b := strconv.Itoa(line.Begin)
		e := strconv.Itoa(line.End)
		text := fmt.Sprintf("%3s-%-3s| %s ", b, e, line.Dump())
		fields = append(fields, text)
	}
	str := strings.Join(fields, "\n")

	return str
}

// AddLine add a line to this page
func (p *Page) AddLine(line *Line) {
	p.Lines = append(p.Lines, line)
}
