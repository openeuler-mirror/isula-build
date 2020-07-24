// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-03-20
// Description: dockerfile related constants

package dockerfile

const (
	noBaseImage = "scratch"

	defaultPathEnv = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	defaultShell = "/bin/sh"

	// refer to pkg/docker/types.go, following are HealthConfig Test type options
	// "NONE": disable healthcheck
	// "CMD-SHELL": run command with system's default shell
	healthCheckTestDisable   = "NONE"
	healthCheckTestTypeShell = "CMD-SHELL"
)
