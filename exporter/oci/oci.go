// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2020-12-15
// Description: oci exporter related functions

package oci

import (
	"sync"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
)

func init() {
	exporter.Register(&_ociExporter)
}

type ociExporter struct {
	items map[string]exporter.Bus
	sync.RWMutex
}

var _ociExporter = ociExporter{
	items: make(map[string]exporter.Bus),
}

func (o *ociExporter) Name() string {
	return constant.OCITransport
}

func (o *ociExporter) Init(opts exporter.ExportOptions, src, destSpec string, localStore *store.Store) error {
	srcReference, _, err := image.FindImage(localStore, src)
	if err != nil {
		return errors.Wrapf(err, "find src image: %q failed with transport %q", src, o.Name())
	}
	destReference, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return errors.Wrapf(err, "parse dest spec: %q failed with transport %q", destSpec, o.Name())
	}

	o.Lock()
	o.items[opts.ExportID] = exporter.Bus{
		SrcRef:  srcReference,
		DestRef: destReference,
	}
	o.Unlock()
	return nil
}

func (o *ociExporter) GetSrcRef(exportID string) types.ImageReference {
	o.RLock()
	defer o.RUnlock()

	if _, ok := o.items[exportID]; ok {
		return o.items[exportID].SrcRef
	}

	return nil
}

func (o *ociExporter) GetDestRef(exportID string) types.ImageReference {
	o.RLock()
	defer o.RUnlock()

	if _, ok := o.items[exportID]; ok {
		return o.items[exportID].DestRef
	}

	return nil
}

func (o *ociExporter) Remove(exportID string) {
	o.Lock()
	delete(o.items, exportID)
	o.Unlock()
}
