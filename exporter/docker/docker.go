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

// Package docker is an exporter for docker daemon
package docker

import (
	"sync"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"

	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
)

func init() {
	exporter.Register(&DockerExporter)
}

type dockerExporter struct {
	items map[string]exporter.Bus
	sync.RWMutex
}

// DockerExporter for exporting images in local store to tarball
var DockerExporter = dockerExporter{
	items: make(map[string]exporter.Bus),
}

func (d *dockerExporter) Name() string {
	return "docker"
}

func (d *dockerExporter) Init(opts exporter.ExportOptions, src, destSpec string, localStore *store.Store) error {
	srcReference, _, err := image.FindImage(localStore, src)
	if err != nil {
		return errors.Errorf("find src image: %q failed, got error: %v", src, err)
	}

	destReference, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return errors.Errorf("parse dest spec: %q failed, got error: %v", destSpec, err)
	}

	d.Lock()
	d.items[opts.ExportID] = exporter.Bus{
		SrcRef:  srcReference,
		DestRef: destReference,
	}
	d.Unlock()
	return nil
}

func (d *dockerExporter) GetSrcRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].SrcRef
	}

	return nil
}

func (d *dockerExporter) GetDestRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].DestRef
	}

	return nil
}

func (d *dockerExporter) Remove(exportID string) {
	d.Lock()
	delete(d.items, exportID)
	d.Unlock()
}
