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
// Create: 2020-07-20
// Description: container transport related functions tests

package container

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/image/v5/docker/reference"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

var (
	mStore    *store.Store
	buildTime time.Time
)

func init() {
	mStore = &store.Store{Store: newMockStore()}
	buildTime = time.Now()
}

func TestNewImage(t *testing.T) {
	type testcase struct {
		name       string
		metadata   *ReferenceMetadata
		localStore *store.Store
		exporting  bool
		isErr      bool
		errStr     string
	}
	referenceName, _ := reference.WithName("busybox:latest")
	var testcases = []testcase{
		{
			name: "layer_not_know",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte("isula-builder"),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
			},
			exporting: true,
			isErr:     true,
			errStr:    "get build container layers failed",
		},
		{
			name: "history",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     "aacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  true,
			isErr:      true,
			errStr:     "history lists",
		},
		{
			name: "normal_with_emptylayer",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				EmptyLayer:  true,
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  true,
			isErr:      false,
		},
		{
			name: "normal_with_no_emptylayer",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  true,
			isErr:      false,
		},
		{
			name: "normal_with_no_emptylayer",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     HasDigestParentlayerID,
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  false,
			isErr:      true,
			errStr:     "history lists",
		},
		{
			name: "bep_change_time",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     HasDigestParentlayerID,
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  false,
			isErr:      true,
			errStr:     "history lists",
		},
		{
			name: "normal_append_history",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     HasDigestParentlayerID,
				BuildTime:   &buildTime,
				PostEmptyLayers: []v1.History{
					{
						Created:    &buildTime,
						CreatedBy:  "/bin/sh",
						Author:     "isula-builder",
						Comment:    "append history test",
						EmptyLayer: false,
					},
					{
						Created:    &buildTime,
						CreatedBy:  "/bin/sh",
						Author:     "isula-builder",
						Comment:    "append history test",
						EmptyLayer: true,
					},
				},
			},
			localStore: mStore,
			exporting:  false,
		},
		{
			name: "analyze_layer_failed",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":6144,"digest":"sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4"}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     HasParentlayerID,
				BuildTime:   &buildTime,
				PostEmptyLayers: []v1.History{
					{
						Created:    &buildTime,
						CreatedBy:  "/bin/sh",
						Author:     "isula-builder",
						Comment:    "append history test",
						EmptyLayer: false,
					},
					{
						Created:    &buildTime,
						CreatedBy:  "/bin/sh",
						Author:     "isula-builder",
						Comment:    "append history test",
						EmptyLayer: true,
					},
				},
			},
			localStore: mStore,
			exporting:  false,
			isErr:      true,
			errStr:     "uncompressed digest is empty",
		},
		{
			name: "analyze_layer_failed",
			metadata: &ReferenceMetadata{
				Name:        referenceName,
				CreatedBy:   "isula",
				Dconfig:     []byte(`{"schemaVersion":"2",MediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1150,"digest":"sha256:1f0cad24dfb44530affe3f5dd8d2550d57f273ec7b88ac060b47a06e051af468"},"layers":[{"mediaType":"application/vnd.docker.container.image.v1+json","size":1441280,"digest":"sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a"}}]}`),
				ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
				LayerID:     HasParentlayerID,
				BuildTime:   &buildTime,
			},
			localStore: mStore,
			exporting:  false,
			isErr:      true,
			errStr:     "invalid character",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.localStore == nil {
				ctxDir := fs.NewDir(t, "store")
				defer func() {
					unix.Unmount(ctxDir.Join("data", "overlay"), 0)
					ctxDir.Remove()
				}()

				dataRoot := filepath.Join(ctxDir.Path(), "/data")
				runRoot := filepath.Join(ctxDir.Path(), "/run")
				store.SetDefaultStoreOptions(store.DaemonStoreOptions{
					DataRoot: dataRoot,
					RunRoot:  runRoot,
				})
				sStore, err := store.GetStore()
				assert.NilError(t, err, tc.name)
				if err != nil {
					defer sStore.Shutdown(false)
				}
				tc.localStore = &sStore
			}
			cf := NewContainerReference(tc.localStore, tc.metadata, tc.exporting)
			ctx := context.TODO()
			buildDirCtx := fs.NewDir(t, t.Name(), fs.WithDir("layer", fs.WithDir("diff", fs.WithFile("diff-file", "diff-file-content"))))
			defer buildDirCtx.Remove()
			buildDir := buildDirCtx.Join("layer", "blob")
			MountPoint = buildDir
			ctx = context.WithValue(ctx, util.BuildDirKey(util.BuildDir), buildDir)
			sc := image.GetSystemContext()
			_, err := cf.NewImage(ctx, sc)
			assert.Equal(t, tc.isErr, err != nil)
			if tc.isErr {
				assert.ErrorContains(t, err, tc.errStr)
			}
		})
	}
}
