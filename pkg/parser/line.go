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
// Description: line related functions

package parser

import "strings"

// Line is the description of a actural effective line in the Dockefile
type Line struct {
	Cells   []*Cell           // cells of the line
	Begin   int               // begin number of the line in the physical Dockefile
	End     int               // end number of the line in the physical Dockefile
	Command string            // command of the line, used upper format here
	Raw     string            // the raw content of the line besides command
	Flags   map[string]string // carries flags for the line, which owns by the Dockerfile CMD
}

// AddCell add a cell to the line
func (l *Line) AddCell(cell *Cell) {
	l.Cells = append(l.Cells, cell)
}

// Dump dumps this line with a string
func (l *Line) Dump() string {
	fields := make([]string, 0, len(l.Cells))
	fields = append(fields, "("+l.Command+")")
	for _, cell := range l.Cells {
		field := cell.dump()
		fields = append(fields, field)
	}
	str := strings.Join(fields, " ")

	return str
}

// IsJSONArgs returns true if this line has the json format args
func (l *Line) IsJSONArgs() bool {
	if l.Flags == nil {
		return false
	}
	if l.Flags["attribute"] == "json" {
		return true
	}

	return false
}
