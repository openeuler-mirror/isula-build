// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Feiyu Yang
// Create: 2020-01-20
// Description: mock store related functions

package container

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/containers/storage"
)

const (
	NoParentlayerID        = "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23"
	HasParentlayerID       = "ddddda0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23"
	ParentLayerID          = "aaaaaa0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23"
	HasDigestParentlayerID = "ccccda0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23"
	ParentDigestLayerID    = "eeeeaa0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23"
)

var (
	MountPoint string
)

type mockStore struct {
	storage.Store
}

func newMockStore() *mockStore {
	return &mockStore{}
}

func (m *mockStore) CreateContainer(id string, names []string, image, layer, metadata string, options *storage.ContainerOptions) (*storage.Container, error) {
	return &storage.Container{
		ID: "abcdedfg",
	}, nil
}

func (m *mockStore) Diff(from, to string, options *storage.DiffOptions) (io.ReadCloser, error) {
	r := strings.NewReader("isula-builder")
	return ioutil.NopCloser(r), nil
}

func (m *mockStore) Layer(id string) (*storage.Layer, error) {
	if id == NoParentlayerID || id == ParentDigestLayerID {
		return &storage.Layer{
			UncompressedDigest: "sha256:c705eaa112d36dd0a3f1a6a747015bcccfeaff1c3b0822ae31f0a11ebd4561d4",
		}, nil
	}
	if id == ParentLayerID {
		return &storage.Layer{}, nil
	}
	if id == HasDigestParentlayerID {
		return &storage.Layer{
			Parent:     ParentDigestLayerID,
			MountPoint: MountPoint,
		}, nil
	}
	if id == HasParentlayerID {
		return &storage.Layer{
			Parent:     ParentLayerID,
			MountPoint: MountPoint,
		}, nil
	}
	return nil, nil
}
