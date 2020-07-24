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
// Description: directive related functions

package dockerfile

import (
	"bufio"
	"io"
	"regexp"

	"github.com/pkg/errors"
)

// default escape token
const defaultEscapeToken = "\\"

var (
	tokenEscapeCommand = regexp.MustCompile(`^#[ \t]*escape[ \t]*=[ \t]*(?P<escapechar>.).*$`)
)

// directive is the structure used during a build run to hold the state of
// parsing directives.
type directive struct {
	escapeToken byte
}

// setEscapeToken sets the token for escaping characters in a Dockerfile.
func (d *directive) setEscapeToken(s string) error {
	if s != "`" && s != "\\" {
		return errors.Errorf("invalid escape token: '%s', must be ` or \\", s)
	}
	d.escapeToken = s[0]

	return nil
}

// newDirective create a directive to parse dockerfile Directive
func newDirective(r io.Reader) (*directive, error) {
	d := &directive{
		escapeToken: defaultEscapeToken[0],
	}

	scanner := bufio.NewScanner(r)
	findEscapeToken := false
	for scanner.Scan() {
		line := scanner.Text()
		matches := tokenEscapeCommand.FindStringSubmatch(line)
		// if first line not match, return the default directive
		if len(matches) == 0 {
			return d, nil
		}

		for i, n := range tokenEscapeCommand.SubexpNames() {
			if n != "escapechar" {
				continue
			}
			if findEscapeToken {
				return nil, errors.New("only support one escape directive")
			}
			findEscapeToken = true
			if err := d.setEscapeToken(matches[i]); err != nil {
				return nil, err
			}
		}
	}

	return d, nil
}
