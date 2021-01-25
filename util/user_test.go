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
// Create: 2020-04-01
// Description: user related common functions tests

package util

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
)

func TestGetChownOptions(t *testing.T) {
	type testCase struct {
		name      string
		chown     string
		UIDWanted int
		GIDWanted int
		isErr     bool
	}
	mountpoint := fmt.Sprintf("/tmp/mount-%d", rand.Int())
	err := os.MkdirAll(mountpoint+"/etc", constant.DefaultSharedDirMode)
	assert.NilError(t, err)
	pFile, err := os.Create(mountpoint + "/etc/passwd")
	if pFile != nil {
		_, err = pFile.WriteString("root:x:0:0:root:/root:/bin/ash\nbin:x:1:1:bin:/bin:/sbin/nologin\n" +
			"daemon:x:2:2:daemon:/sbin:/sbin/nologin\n555555:x:3:4:adm:/var/adm:/sbin/nologin\n" +
			"ok:x:211:211:daemon:/sbin:/sbin/nologin")
		assert.NilError(t, err)
		pFile.Close()
	}

	gFile, err := os.Create(mountpoint + "/etc/group")
	if gFile != nil {
		_, err = gFile.WriteString("root:x:0:root\nbin:x:1:root,bin,daemon\n" +
			"daemon:x:2:root,bin,daemon\n77777:x:3:root,bin,adm\n")
		assert.NilError(t, err)
		gFile.Close()
	}

	cases := []testCase{
		{
			name:      "1",
			chown:     "",
			UIDWanted: 0,
			GIDWanted: 0,
		},
		{
			name:      "2",
			chown:     "1555",
			UIDWanted: 1555,
			GIDWanted: 1555,
		},
		{
			name:      "3",
			chown:     "555555",
			UIDWanted: 3,
			GIDWanted: 555555,
		},
		{
			name:      "4",
			chown:     "5ab2",
			UIDWanted: 0,
			GIDWanted: 0,
			isErr:     true,
		},
		{
			name:      "5",
			chown:     "52222222222222222222222222222",
			UIDWanted: 0,
			GIDWanted: 0,
			isErr:     true,
		},
		{
			name:      "6",
			chown:     "0:0",
			UIDWanted: 0,
			GIDWanted: 0,
		},
		{
			name:      "7",
			chown:     "111:112",
			UIDWanted: 111,
			GIDWanted: 112,
		},
		{
			name:      "8",
			chown:     "555555:555555",
			UIDWanted: 3,
			GIDWanted: 555555,
		},
		{
			name:      "9",
			chown:     "5555552:77777",
			UIDWanted: 5555552,
			GIDWanted: 3,
		},
		{
			name:      "10",
			chown:     "5555552:77777777777777777777777777777777777",
			UIDWanted: 0,
			GIDWanted: 0,
			isErr:     true,
		},
		{
			name:      "11",
			chown:     "daemon",
			UIDWanted: 2,
			GIDWanted: 2,
		},
		{
			name:      "12",
			chown:     "root:77777",
			UIDWanted: 0,
			GIDWanted: 3,
		},
		{
			name:      "13",
			chown:     "root:bin",
			UIDWanted: 0,
			GIDWanted: 1,
		},
		{
			name:      "14",
			chown:     "root:bin2",
			UIDWanted: 0,
			GIDWanted: 0,
			isErr:     true,
		},
		{
			name:      "15",
			chown:     "ok",
			UIDWanted: 0,
			GIDWanted: 0,
			isErr:     true,
		},
		{
			name:      "16",
			chown:     "77777",
			UIDWanted: 77777,
			GIDWanted: 3,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pair, err := GetChownOptions(c.chown, mountpoint)
			assert.Equal(t, err != nil, c.isErr)
			assert.Equal(t, pair.UID, c.UIDWanted)
			assert.Equal(t, pair.GID, c.GIDWanted)
		})
	}

	err = os.RemoveAll(mountpoint)
	assert.NilError(t, err)
}
