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
// Description: parser related functions

package parser

import (
	"io"

	"github.com/pkg/errors"
)

// DefaultParser is default parser name with 'dockerfile'
const DefaultParser = "dockerfile"

var parsers map[string]Parser

// Parser is an interface to implement a Dockerfile parser
type Parser interface {
	Parse(r io.Reader, onbuild bool) (*PlayBook, error)
	ParseIgnore(dir string) ([]string, error)
}

// Register registers a parse to the parsers
func Register(name string, parser Parser) {
	if parsers == nil {
		parsers = make(map[string]Parser)
	}
	if _, ok := parsers[name]; ok {
		return
	}
	parsers[name] = parser
}

// NewParser creates a parse with the given name
func NewParser(name string) (Parser, error) {
	if name == "" {
		name = DefaultParser
	}
	if _, ok := parsers[name]; !ok {
		return nil, errors.Errorf("parser %s not support", name)
	}
	return parsers[name], nil
}
