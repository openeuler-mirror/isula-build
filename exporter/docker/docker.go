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
// Description: docker repository exporter related functions

// Package daemon is an exporter for docker daemon
package daemon

import (
	"github.com/containers/image/v5/types"

	"isula.org/isula-build/exporter"
)

func init() {
	exporter.Register(&_dockerExporter)
}

type dockerExporter struct {
	exporter.Bus
}

var _dockerExporter = dockerExporter{}

func (d *dockerExporter) Name() string {
	return "docker"
}

func (d *dockerExporter) Init(src, dest types.ImageReference) {
	d.SrcRef = src
	d.DestRef = dest
}

func (d *dockerExporter) GetSrcRef() types.ImageReference {
	return d.SrcRef
}

func (d *dockerExporter) GetDestRef() types.ImageReference {
	return d.DestRef
}
