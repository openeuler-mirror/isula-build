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
// Description: cell related functions

// Package parser includes docker file parser relate struct and functions
package parser

import (
	"strings"
)

// Cell is the description of a token in the Dockerfile
type Cell struct {
	Value string // token value
}

// dump cell's content
func (c *Cell) dump() string {
	fields := []string{c.Value}
	str := strings.Join(fields, " ")
	str = "(" + str + ")"

	return str
}
