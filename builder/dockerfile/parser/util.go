// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jingxiao Lu
// Create: 2020-03-20
// Description: dockerfile parse common functions

package dockerfile

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	// Split ${param} or $param.
	// param should start with char
	regParam = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*`)
	// regParamSpecial use to match ${variable:-word} and ${variable:+word}
	regParamSpecial = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9\-\+_:]+`)
)

const (
	dollar      = '$'
	singleQuote = '\''
	doubleQuote = '"'
	leftBrace   = '{'
	rightBrace  = '}'
	rightSlash  = '\\'
	// form ${variable:-word}: if variable not defined, return word
	bashModifier1 = ":-"
	// form ${variable:+word}: if variable defined, return word; otherwise return ""
	bashModifier2 = ":+"
)

type resolver struct {
	origin      string              // string to resolveFunc
	resolved    string              // string resolved, will be returned to caller
	resolveFunc func(string) string // func to resolveFunc args

	length       int  // length of input string
	idx          int  // searching idx
	strict       bool // strict mode (for FROM command) or easy mode
	singleQuotes bool // true means the searching meets '\'' and matcher is handling the string after it
	doubleQuotes bool // true means the searching meets '"' and matcher is handling the string after it
}

// ResolveParam inputs ${param}, returns param
// inputs ${param}_${param2}, returns param_param2
// strict bool (for FROM), if true, the arg must matched, otherwise return err; if false, return ""
// resolveArg isn't common, since builder is managing unusedArgs, so a func from builder may be needed
func ResolveParam(s string, strict bool, resolveArg func(string) string) (string, error) {
	r := &resolver{
		origin:      s,
		strict:      strict,
		resolveFunc: resolveArg,
		length:      len(s),
	}

	for r.idx <= r.length {
		// searching end
		if r.idx == r.length {
			if err := r.searchingEnd(); err != nil {
				return "", err
			}
			break
		}

		if r.origin[r.idx] != dollar {
			r.noDollar()
			continue
		}

		// next resolving the arg after '$'
		r.idx++
		if r.idx == r.length {
			return "", errors.Errorf("ending with single %q", dollar)
		}

		// ok, let's see if it is an arg
		// next is the form without braces: $arg
		if r.origin[r.idx] != leftBrace {
			if err := r.noBrace(); err != nil {
				return "", err
			}
			continue
		}

		if err := r.inBrace(); err != nil {
			return "", err
		}
	}
	return r.resolved, nil
}

func (r *resolver) searchingEnd() error {
	if r.singleQuotes {
		return errors.New("fails to get another single-quotes")
	}
	if r.doubleQuotes {
		return errors.New("fails to get another double-quotes")
	}
	return nil
}

func (r *resolver) noDollar() {
	// case ''
	if r.origin[r.idx] == singleQuote {
		r.singleQuotes = !r.singleQuotes
		r.idx++
		return
	}
	// case ""
	if r.origin[r.idx] == doubleQuote {
		r.doubleQuotes = !r.doubleQuotes
		r.idx++
		return
	}

	// case "\$foo". it will be form as "\\$foo"
	if r.origin[r.idx] == rightSlash && r.idx+1 < r.length && r.origin[r.idx+1] == dollar {
		r.resolved += r.origin[r.idx : r.idx+2]
		const tokenLen = 2 // tokens are "\\$"
		r.idx += tokenLen
		return
	}

	// not "\$", this must be hyphen between args, such as '/' in "hub/image" or '_' in 'module_arch'
	r.resolved += string(r.origin[r.idx])
	r.idx++
	return
}

func (r *resolver) noBrace() error {
	// it seems this arg is with form "$arg"
	// try to match an arg with regexp
	argIdx := regParam.FindStringIndex(r.origin[r.idx:])
	if argIdx == nil {
		if r.strict {
			return errors.Errorf("no valid arg after %q", dollar)
		}
		r.idx++
		return nil
	}
	// re-calc the argIdx with idx
	argIdx[0], argIdx[1] = argIdx[0]+r.idx, argIdx[1]+r.idx

	arg := r.origin[argIdx[0]:argIdx[1]]
	val := r.resolveFunc(arg)
	if len(val) == 0 && r.strict {
		return errors.Errorf("no matching key for arg %q", arg)
	}
	r.resolved += val

	// move idx to next char
	r.idx = argIdx[1]
	return nil
}

func (r *resolver) inBrace() error {
	// ok, we got a '{', this arg must be enclosed by ${} with form "${arg}"
	r.idx++
	if r.idx == r.length || r.origin[r.idx] == rightBrace {
		// idx+1 == len(s) means encounters: ${
		// s[idx+1] == '}' means encounters: ${}
		return errors.New("an arg is expected after `${`")
	}

	// try to match an arg with regexp
	sub := r.origin[r.idx:]
	var argIdx []int
	// for type ${variable:-word} or ${variable:+word}
	// the whole string will be as arg put to resolveArg()
	// and be split with ":-" or ":+" before handling in resolveArg()
	if strings.Contains(sub, bashModifier1) || strings.Contains(sub, bashModifier2) {
		argIdx = regParamSpecial.FindStringIndex(sub)
	} else {
		argIdx = regParam.FindStringIndex(sub)
	}
	if argIdx == nil {
		return errors.Errorf("no valid arg after `${`")
	}
	// re-calc the argIdx with idx
	argIdx[0], argIdx[1] = argIdx[0]+r.idx, argIdx[1]+r.idx

	arg := r.origin[argIdx[0]:argIdx[1]]
	val := r.resolveFunc(arg)
	// ${val:+word} can also returns "" if val is not defined. so please ignore it here
	if len(val) == 0 && !strings.Contains(arg, bashModifier2) && r.strict {
		return errors.Errorf("no matching key for arg %q", arg)
	}
	r.resolved += val

	// after handle arg and value, move to next char, it must be '}'
	r.idx = argIdx[1]
	if r.idx >= r.length || r.origin[r.idx] != rightBrace {
		return errors.Errorf("no ending `}` after `${` with arg %q", arg)
	}

	// after ending '}', move to next char again
	r.idx++
	return nil
}
