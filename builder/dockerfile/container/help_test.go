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
// Description: container transport related functions tests

package container

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/storage/pkg/reexec"
	"github.com/opencontainers/go-digest"
	"gotest.tools/assert"

	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/store"
)

const (
	interval = 1000000000 * 60 * 5
	timeout  = 3000000000
)

var (
	localStore store.Store
	rootDir    = "/tmp/isula-build/container"
)

func init() {
	reexec.Init()
	dataRoot := rootDir + "/data"
	runRoot := rootDir + "/run"
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: dataRoot,
		RunRoot:  runRoot,
	})
	localStore, _ = store.GetStore()
}

func TestMain(m *testing.M) {
	fmt.Println("container package test begin")
	m.Run()
	fmt.Println("container package test end")
	clean()
}

func clean() {
	localStore.Shutdown(false)
	os.RemoveAll(rootDir)
}

func TestCreateConfigsAndManifests(t *testing.T) {
	var name reference.Named
	metadata := &ReferenceMetadata{
		Name:        name,
		CreatedBy:   "isula",
		Dconfig:     []byte(`{"created":"2017-05-12T21:36:57.851043Z","container":"e8b1134b0d5017566d417c25b8dc050e56c06b19e29fda15fddad04e1b4adc2d","container_config":{"Hostname":"ab281de98ba0","Domainname":"","User":"","Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["sh"],"Healthcheck":{"Test":["CMD-SHELL","curl -f http://localhost/ || exit 1"],"Interval":300000000000,"Timeout":3000000000},"Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{}},"docker_version":"1.12.1","config":{"Hostname":"ab281de98ba0","Domainname":"","User":"","Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["sh"],"Healthcheck":{"Test":["CMD-SHELL","curl -f http://localhost/ || exit 1"],"Interval":300000000000,"Timeout":3000000000},"Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{}},"architecture":"arm64","os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:f91599b3986be816a2c74a3c05fda82e1b29f55eb12bc2b54fbfdfc3b5773edc"]},"history":[{"created":"2017-05-12T21:36:57.08197Z","created_by":"/bin/sh -c #(nop) ADD file:e9e6f86057e43a27b678a139b906091c3ecb1600b08ad17e80ff5ad56920c96e in / "},{"created":"2017-05-12T21:36:57.851043Z","created_by":"/bin/sh -c #(nop)  CMD [\"sh\"]","empty_layer":true}]}`),
		ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
		LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
	}

	containerRef := NewContainerReference(localStore, metadata, false)
	dimage, dmanifest, err := containerRef.createConfigsAndManifests()
	assert.NilError(t, err)
	assert.DeepEqual(t, dimage, docker.Image{
		V1Image: docker.V1Image{
			ID:        "",
			Parent:    "",
			Comment:   "",
			Created:   containerRef.created, //"2020-04-15 07:41:47.96447546 +0000 UTC",
			Container: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
			ContainerConfig: docker.Config{
				Hostname:     "ab281de98ba0",
				User:         "",
				ExposedPorts: nil,
				Env:          []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
				Cmd:          []string{"sh"},
				Healthcheck: &docker.HealthConfig{
					Test:     []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
					Interval: time.Duration(interval),
					Timeout:  time.Duration(timeout),
				},
				Volumes:     nil,
				WorkingDir:  "",
				Entrypoint:  nil,
				OnBuild:     nil,
				Labels:      map[string]string{},
				StopSignal:  "",
				StopTimeout: nil,
				Shell:       nil,
			},
			DockerVersion: "1.12.1",
			Author:        "",
			Config: &docker.Config{
				Hostname: "ab281de98ba0",
				Env:      []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
				Cmd:      []string{"sh"},
				Healthcheck: &docker.HealthConfig{
					Test:     []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
					Interval: time.Duration(interval),
					Timeout:  time.Duration(timeout),
				},
				Labels: map[string]string{},
			},
			Architecture: "arm64",
			OS:           "linux",
			Size:         0,
		},
		Parent: "",
		RootFS: &docker.RootFS{Type: "layers", DiffIDs: []digest.Digest{}},
		History: []docker.History{
			{
				Created:   time.Date(2017, 5, 12, 21, 36, 57, 81970000, time.UTC), //created, //"2017-05-12 21:36:57.08197 +0000 UTC",
				CreatedBy: "/bin/sh -c #(nop) ADD file:e9e6f86057e43a27b678a139b906091c3ecb1600b08ad17e80ff5ad56920c96e in / ",
			},
			{
				Created:    time.Date(2017, 5, 12, 21, 36, 57, 851043000, time.UTC), //"2017-05-12 21:36:57.851043 +0000 UTC",
				CreatedBy:  `/bin/sh -c #(nop)  CMD ["sh"]`,
				EmptyLayer: true,
			},
		},
	})

	assert.DeepEqual(t, dmanifest, docker.Manifest{
		Versioned: docker.Versioned{
			SchemaVersion: schemaVersion,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		},
		Config: docker.Descriptor{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Size:      0,
			Digest:    "",
			URLs:      nil,
		},
		Layers: []docker.Descriptor{},
	})
}

func TestPrepareRWLayers(t *testing.T) {
	var name reference.Named
	metadata := &ReferenceMetadata{
		Name:        name,
		CreatedBy:   "isula",
		Dconfig:     []byte("isula-builder"),
		ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
		LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
	}

	containerRef := NewContainerReference(localStore, metadata, false)
	_, err := containerRef.getContainerLayers()
	assert.ErrorContains(t, err, "unable to read layer")
}
