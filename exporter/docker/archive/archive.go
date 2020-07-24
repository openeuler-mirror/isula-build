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
// Description: archive exporter related functions

// Package archive is used to export archive type images
package archive

import (
	"github.com/containers/image/v5/types"

	"isula.org/isula-build/exporter"
)

func init() {
	exporter.Register(&_dockerArchiveExporter)
}

type dockerArchiveExporter struct {
	exporter.Bus
}

var _dockerArchiveExporter = dockerArchiveExporter{}

func (d *dockerArchiveExporter) Name() string {
	return "docker-archive"
}

func (d *dockerArchiveExporter) Init(src, dest types.ImageReference) {
	d.SrcRef = src
	d.DestRef = dest
}

func (d *dockerArchiveExporter) GetSrcRef() types.ImageReference {
	return d.SrcRef
}

func (d *dockerArchiveExporter) GetDestRef() types.ImageReference {
	return d.DestRef
}
