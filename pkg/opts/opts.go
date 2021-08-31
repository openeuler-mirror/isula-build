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
// Create: 2020-06-30
// Description: This file is used to define a custom flag for store listOpts

// Package opts provides interface for managing ListOpts
package opts

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ValidatorFunc is a validate function used to check the value
type validatorFunc func(value string) (string, string, error)

// ListOpts holds a list of values
type ListOpts struct {
	Values    map[string]string
	validator validatorFunc
}

// Set appends the value to the opts
func (opts *ListOpts) Set(value string) error {
	if opts.validator != nil {
		k, v, err := opts.validator(value)
		if err != nil {
			return err
		}
		opts.Values[k] = v
	}

	return nil
}

// String returns the values the opts
func (opts *ListOpts) String() string {
	s := make([]string, 0, len(opts.Values))
	for k, v := range opts.Values {
		kv := fmt.Sprintf("%s=%s", k, v)
		s = append(s, kv)
	}

	return strings.Join(s, ",")
}

// Type returns the type of the opts
func (opts *ListOpts) Type() string {
	return "listopts"
}

// NewListOpts creates a new ListOpts
func NewListOpts(validator validatorFunc) ListOpts {
	values := make(map[string]string)
	return ListOpts{
		Values:    values,
		validator: validator,
	}
}

// OptValidator validates the value and return a key, value pair
func OptValidator(value string) (string, string, error) {
	const partsNum = 2
	kv := strings.SplitN(value, "=", partsNum)
	if len(kv) < partsNum {
		return "", "", errors.New("invalid format, option need a key=value pair")
	}

	return kv[0], kv[1], nil
}
