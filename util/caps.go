// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Feiyu Yang
// Create: 2020-08-15
// Description: This file provides check func for linux capability.

package util

import (
	"strings"

	"github.com/syndtr/gocapability/capability"
)

var caps map[string]capability.Cap

func init() {
	caps = make(map[string]capability.Cap)
	for _, c := range capability.List() {
		caps["CAP_"+strings.ToUpper(c.String())] = c
	}
}

// CheckCap checks if cap is valid
func CheckCap(c string) bool {
	_, ok := caps[c]
	return ok
}
