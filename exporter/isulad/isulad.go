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
// Description: isulad exporter related functions

// Package isulad is used to export images to isulad
package isulad

import (
	"sync"

	"github.com/containers/image/v5/types"

	"isula.org/isula-build/exporter"
)

func init() {
	exporter.Register(&_isuladExporter)
}

type isuladExporter struct {
	items map[string]exporter.Bus
	sync.RWMutex
}

var _isuladExporter = isuladExporter{
	items: make(map[string]exporter.Bus),
}

func (d *isuladExporter) Name() string {
	return "isulad"
}

func (d *isuladExporter) Init(exportID string, src, dest types.ImageReference) {
	d.Lock()
	d.items[exportID] = exporter.Bus{
		SrcRef:  src,
		DestRef: dest,
	}
	d.Unlock()
}

func (d *isuladExporter) GetSrcRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].SrcRef
	}

	return nil
}

func (d *isuladExporter) GetDestRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].DestRef
	}

	return nil
}

func (d *isuladExporter) Remove(exportID string) {
	d.Lock()
	delete(d.items, exportID)
	d.Unlock()
}
