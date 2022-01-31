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
// Description: user related common functions

package util

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containers/storage/pkg/idtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
)

var errNoSuchUser = errors.New("no such user")

const (
	etcPasswdColumnNum = 7
	etcGroupColumnNum  = 4
)

// GetChownOptions is used to get chown options
func GetChownOptions(chown, mountpoint string) (idtools.IDPair, error) {
	mountpoint = filepath.Clean(mountpoint)
	if len(chown) == 0 {
		return idtools.IDPair{}, nil
	}
	pairID := strings.Split(chown, ":")
	const pairLen = 2
	// if chown is "a", gid should also be "a", then pairID will be "a a"
	if len(pairID) < pairLen {
		pairID = append(pairID, pairID[0])
	}

	u, err := searchUserGroup(pairID[0], mountpoint+"/etc/passwd", true)
	if err != nil && err != errNoSuchUser {
		return idtools.IDPair{}, errors.Wrapf(err, "failed to search user in /etc/passwd")
	}
	if err == errNoSuchUser {
		u, err = strconv.Atoi(pairID[0])
		if err != nil {
			return idtools.IDPair{}, errors.Wrapf(err, "failed to convert string uid to int uid")
		}
	}

	g, err := searchUserGroup(pairID[1], mountpoint+"/etc/group", false)
	if err != nil && err != errNoSuchUser {
		return idtools.IDPair{}, errors.Wrapf(err, "failed to search user in /etc/group")
	}
	if err == errNoSuchUser {
		g, err = strconv.Atoi(pairID[1])
		if err != nil {
			return idtools.IDPair{}, errors.Wrapf(err, "failed to convert string gid to int gid")
		}
	}

	logrus.Debugf("UID is %v and GID is %v", u, g)
	return idtools.IDPair{UID: u, GID: g}, nil
}

// searchUserGroup searches user in etc/passwd and group in etc/group
// function caller should make sure the path is clean
func searchUserGroup(name, path string, userFlag bool) (int, error) {
	if err := CheckFileInfoAndSize(path, constant.MaxFileSize); err != nil {
		return 0, err
	}
	f, err := os.Open(path) // nolint:gosec
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			logrus.Warningf("Closing %q failed: %v", path, cerr)
		}
	}()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		info := strings.Split(line, ":")
		if info[0] != name {
			continue
		}

		// /etc/passwd, xxx:x:0:0:xxx:/xxx:/bin/xxx, the length is 7
		// /etc/group, xx:x:0:, the length is 4
		if (userFlag && len(info) != etcPasswdColumnNum) || (!userFlag && len(info) != etcGroupColumnNum) {
			continue
		}

		const indexID = 2
		userGroupID, err := strconv.Atoi(info[indexID])
		if err != nil {
			return 0, err
		}
		return userGroupID, nil
	}

	return 0, errNoSuchUser
}
