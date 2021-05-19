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
// Description: oci-archive exporter related functions

package archive

import (
	"fmt"
	"strings"
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
	exporter.Register(&_ociArchiveExporter)
}

type ociArchiveExporter struct {
	items map[string]exporter.Bus
	sync.RWMutex
}

// OCIArchiveExporter for exporting images from local store to tarball in oci format
var _ociArchiveExporter = ociArchiveExporter{
	items: make(map[string]exporter.Bus),
}

func (o *ociArchiveExporter) Name() string {
	return constant.OCIArchiveTransport
}

func (o *ociArchiveExporter) Init(opts exporter.ExportOptions, src, destSpec string, localStore *store.Store) error {
	// Same as docker archive file, it needs to use ImageID to get reference.
	// As a result, docker.io/library/ will not added to reference domain.
	srcReference, _, err := image.FindImage(localStore, src)
	if err != nil {
		return errors.Wrapf(err, "find src image: %q failed with transport %q", src, o.Name())
	}
	// Add name:tag to oci-archive index.json file for save, make sure image saved with a tag
	if strings.Contains(src, ":") {
		destSpec = fmt.Sprintf("%s:%s", destSpec, src)
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

func (o *ociArchiveExporter) GetSrcRef(exportID string) types.ImageReference {
	o.RLock()
	defer o.RUnlock()

	if _, ok := o.items[exportID]; ok {
		return o.items[exportID].SrcRef
	}

	return nil
}

func (o *ociArchiveExporter) GetDestRef(exportID string) types.ImageReference {
	o.RLock()
	defer o.RUnlock()

	if _, ok := o.items[exportID]; ok {
		return o.items[exportID].DestRef
	}

	return nil
}

func (o *ociArchiveExporter) Remove(exportID string) {
	o.Lock()
	delete(o.items, exportID)
	o.Unlock()
}
