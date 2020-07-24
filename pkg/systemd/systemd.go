// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2020-03-20
// Description: systemd related functions

// Package systemd is used to notify systemd
package systemd

import (
	"os"

	systemdDaemon "github.com/coreos/go-systemd/daemon"
	"github.com/sirupsen/logrus"
)

// NotifySystemReady notifies host that the server is booted up
func NotifySystemReady() {
	if os.Getenv("NOTIFY_SOCKET") != "" {
		notified, notifyErr := systemdDaemon.SdNotify(false, systemdDaemon.SdNotifyReady)
		logrus.Debugf("SdNotifyReady notified=%v, err=%v", notified, notifyErr)
	}
}

// NotifySystemStopping notifies host that the server is stopped
func NotifySystemStopping() {
	if os.Getenv("NOTIFY_SOCKET") != "" {
		notified, notifyErr := systemdDaemon.SdNotify(false, systemdDaemon.SdNotifyStopping)
		logrus.Debugf("SdNotifyStopping notified=%v, err=%v", notified, notifyErr)
	}
}
