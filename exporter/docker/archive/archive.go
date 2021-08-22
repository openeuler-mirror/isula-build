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
	"strings"
	"sync"

	"github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
)

func init() {
	exporter.Register(&DockerArchiveExporter)
}

type dockerArchiveExporter struct {
	itemsArchiveWriter map[string]*archive.Writer
	items              map[string]exporter.Bus
	sync.RWMutex
}

// DockerArchiveExporter for exporting images in local store to tarball
var DockerArchiveExporter = dockerArchiveExporter{
	items:              make(map[string]exporter.Bus),
	itemsArchiveWriter: make(map[string]*archive.Writer),
}

func (d *dockerArchiveExporter) Name() string {
	return constant.DockerArchiveTransport
}

func (d *dockerArchiveExporter) Init(opts exporter.ExportOptions, src, destSpec string, localStore *store.Store) error {
	var (
		srcReference  types.ImageReference
		destReference types.ImageReference
		err           error
	)
	const partsNum = 2
	// src could be form of ImageID digest or name[:tag]
	// destSpec could be "file:name:tag" or "file:name" or just "file" with transport "docker-archive", such as docker-archive:output.tar:name:tag
	// When more than two parts, build must be called
	if parts := strings.Split(destSpec, ":"); len(parts) > partsNum {
		srcReference, _, err = image.FindImageLocally(localStore, src)
		if err != nil {
			return errors.Wrapf(err, "find src image: %q failed with transport %q", src, d.Name())
		}
		destReference, err = alltransports.ParseImageName(destSpec)
		if err != nil {
			return errors.Wrapf(err, "parse dest spec: %q failed with transport %q", destSpec, d.Name())
		}
		d.Lock()
		d.items[opts.ExportID] = exporter.Bus{
			SrcRef:  srcReference,
			DestRef: destReference,
		}
		d.Unlock()

		return nil
	}

	// from build or save, we can get path from the other part
	archiveFilePath := strings.SplitN(destSpec, ":", partsNum)[1]

	if DockerArchiveExporter.GetArchiveWriter(opts.ExportID) == nil {
		archWriter, wErr := archive.NewWriter(opts.SystemContext, archiveFilePath)
		if wErr != nil {
			return errors.Wrapf(wErr, "create archive writer failed")
		}
		DockerArchiveExporter.InitArchiveWriter(opts.ExportID, archWriter)
	}

	srcReference, _, err = image.FindImageLocally(localStore, src)
	if err != nil {
		return errors.Wrapf(err, "find src image: %q failed with transport %q", src, d.Name())
	}

	archiveWriter := DockerArchiveExporter.GetArchiveWriter(opts.ExportID)
	destReference, err = archiveWriter.NewReference(nil)
	if err != nil {
		return errors.Wrapf(err, "parse dest spec: %q failed", destSpec)
	}

	d.Lock()
	d.items[opts.ExportID] = exporter.Bus{
		SrcRef:  srcReference,
		DestRef: destReference,
	}
	d.Unlock()

	return nil
}
func (d *dockerArchiveExporter) InitArchiveWriter(exportID string, writer *archive.Writer) {
	d.Lock()

	d.itemsArchiveWriter[exportID] = writer

	d.Unlock()
}

func (d *dockerArchiveExporter) GetSrcRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].SrcRef
	}

	return nil
}

func (d *dockerArchiveExporter) GetDestRef(exportID string) types.ImageReference {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.items[exportID]; ok {
		return d.items[exportID].DestRef
	}

	return nil
}

func (d *dockerArchiveExporter) GetArchiveWriter(exportID string) *archive.Writer {
	d.RLock()
	defer d.RUnlock()

	if _, ok := d.itemsArchiveWriter[exportID]; ok {
		return d.itemsArchiveWriter[exportID]
	}

	return nil
}

func (d *dockerArchiveExporter) Remove(exportID string) {
	d.Lock()
	delete(d.items, exportID)
	d.Unlock()
}

func (d *dockerArchiveExporter) RemoveArchiveWriter(exportID string) {
	d.Lock()
	delete(d.itemsArchiveWriter, exportID)
	d.Unlock()
}
