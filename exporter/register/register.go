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
// Description: exporter register

// Package register is used to register exporter
package register

import (
	// register the docker exporter
	_ "isula.org/isula-build/exporter/docker"
	// register the docker-archive exporter
	_ "isula.org/isula-build/exporter/docker/archive"
	// register the docker-daemon exporter
	_ "isula.org/isula-build/exporter/docker/daemon"
	// register the isulad exporter
	_ "isula.org/isula-build/exporter/isulad"
	// register the manifest exporter
	_ "isula.org/isula-build/exporter/manifest"
	// register the oci exporter
	_ "isula.org/isula-build/exporter/oci"
	// register the oci-archive exporter
	_ "isula.org/isula-build/exporter/oci/archive"
)
