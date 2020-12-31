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
	"fmt"
	"strings"
	"sync"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/stringid"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
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

func (d *isuladExporter) Init(opts exporter.ExportOptions, src, destSpec string, localStore *store.Store) error {
	// For isulad, Init is only called from build command.
	// src is form of ImageID digest, destSpec is form of isulad:image:tag
	const partsNum = 2
	parts := strings.SplitN(destSpec, ":", partsNum)
	if len(parts) != partsNum {
		return errors.Errorf(`invalid dest spec %q, expected colon-separated exporter:reference`, destSpec)
	}

	srcReference, _, err := image.FindImage(localStore, src)
	if err != nil {
		return errors.Errorf("find src image: %q failed, got error: %v", src, err)
	}

	randomID := stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]
	isuladTarPath, err := securejoin.SecureJoin(opts.DataDir, fmt.Sprintf("isula-build-tmp-%s.tar", randomID))
	if err != nil {
		return err
	}
	// construct format: transport:path:image:tag
	// parts[1] here could not be empty cause client-end already processed it
	destSpec = fmt.Sprintf("docker-archive:%s:%s", isuladTarPath, parts[1])
	logrus.Infof("Process isulad output %s", destSpec)
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
