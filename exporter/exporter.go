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
// Description: exporter related functions

package exporter

import (
	"github.com/containers/image/v5/types"
)

var exporters map[string]Exporter

// Bus is an struct for SrcRef and DestRef
type Bus struct {
	SrcRef  types.ImageReference
	DestRef types.ImageReference
}

func init() {
	exporters = make(map[string]Exporter)
}

// Exporter is an interface
type Exporter interface {
	Name() string
	Init(src, dest types.ImageReference)
	GetSrcRef() types.ImageReference
	GetDestRef() types.ImageReference
}

// Register register an exporter
func Register(e Exporter) {
	name := e.Name()
	if _, ok := exporters[name]; ok {
		return
	}
	exporters[name] = e
}

// GetAnExporter an Exporter for the given name
func GetAnExporter(name string) Exporter {
	return exporters[name]
}

// IsSupport return a bool whether a exporter is support
func IsSupport(name string) bool {
	_, ok := exporters[name]
	return ok
}
