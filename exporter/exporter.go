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
	"sync"

	"github.com/containers/image/v5/types"

	"isula.org/isula-build/store"
)

const (
	// DockerTransport used to export docker image format images to registry
	DockerTransport = "docker"

	// DockerArchiveTransport used to export docker image format images to local tarball
	DockerArchiveTransport = "docker-archive"

	// DockerDaemonTransport used to export images to docker daemon
	DockerDaemonTransport = "docker-daemon"

	// OCITransport used to export oci image format images to registry
	OCITransport = "oci"

	// OCIArchiveTransport used to export oci image format images to local tarball
	OCIArchiveTransport = "oci-archive"

	// IsuladTransport use to export images to isulad
	IsuladTransport = "isulad"

	// ManifestTransport used to export manifest list
	ManifestTransport = "manifest"
)

type exportHub struct {
	items map[string]Exporter
	sync.RWMutex
}

var hub exportHub

// Bus is an struct for SrcRef and DestRef
type Bus struct {
	SrcRef  types.ImageReference
	DestRef types.ImageReference
}

func init() {
	hub.items = make(map[string]Exporter)
}

// Exporter is an interface
type Exporter interface {
	Name() string
	Init(opts ExportOptions, src, destSpec string, localStore *store.Store) error
	GetSrcRef(exportID string) types.ImageReference
	GetDestRef(exportID string) types.ImageReference
	Remove(exportID string)
}

// Register registers an exporter
func Register(e Exporter) {
	hub.Lock()
	defer hub.Unlock()

	name := e.Name()
	if _, ok := hub.items[name]; ok {
		return
	}
	hub.items[name] = e
}

// GetAnExporter returns an Exporter for the given name
func GetAnExporter(name string) Exporter {
	hub.RLock()
	defer hub.RUnlock()

	return hub.items[name]
}

// IsSupport returns true when the specific exporter is supported
func IsSupport(name string) bool {
	hub.RLock()
	defer hub.RUnlock()

	_, ok := hub.items[name]
	return ok
}
