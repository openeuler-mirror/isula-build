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

package store

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/containers/storage"
	drivers "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/opencontainers/go-digest"
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

// MockStore is the mock of store
type MockStore interface {
	RunRoot() string
	GraphRoot() string
	GraphDriverName() string
	GraphOptions() []string
	UIDMap() []idtools.IDMap
	GIDMap() []idtools.IDMap
	GraphDriver() (drivers.Driver, error)
	CreateLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions) (*storage.Layer, error)
	PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions, diff io.Reader) (*storage.Layer, int64, error)
	CreateImage(id string, names []string, layer, metadata string, options *storage.ImageOptions) (*storage.Image, error)
	CreateContainer(id string, names []string, image, layer, metadata string, options *storage.ContainerOptions) (*storage.Container, error)
	Metadata(id string) (string, error)
	SetMetadata(id, metadata string) error
	Exists(id string) bool
	Status() ([][2]string, error)
	Delete(id string) error
	DeleteLayer(id string) error
	DeleteImage(id string, commit bool) ([]string, error)
	DeleteContainer(id string) error
	Wipe() error
	Mount(id, mountLabel string) (string, error)
	Unmount(id string, force bool) (bool, error)
	Mounted(id string) (int, error)
	Changes(from, to string) ([]archive.Change, error)
	DiffSize(from, to string) (int64, error)
	Diff(from, to string, options *storage.DiffOptions) (io.ReadCloser, error)
	ApplyDiff(to string, diff io.Reader) (int64, error)
	LayersByCompressedDigest(d digest.Digest) ([]storage.Layer, error)
	LayersByUncompressedDigest(d digest.Digest) ([]storage.Layer, error)
	LayerSize(id string) (int64, error)
	LayerParentOwners(id string) ([]int, []int, error)
	Layers() ([]storage.Layer, error)
	Images() ([]storage.Image, error)
	Containers() ([]storage.Container, error)
	Names(id string) ([]string, error)
	SetNames(id string, names []string) error
	ListImageBigData(id string) ([]string, error)
	ImageBigData(id, key string) ([]byte, error)
	ImageBigDataSize(id, key string) (int64, error)
	ImageBigDataDigest(id, key string) (digest.Digest, error)
	SetImageBigData(id, key string, data []byte, digestManifest func([]byte) (digest.Digest, error)) error
	ImageSize(id string) (int64, error)
	ListContainerBigData(id string) ([]string, error)
	ContainerBigData(id, key string) ([]byte, error)
	ContainerBigDataSize(id, key string) (int64, error)
	ContainerBigDataDigest(id, key string) (digest.Digest, error)
	SetContainerBigData(id, key string, data []byte) error
	ContainerSize(id string) (int64, error)
	Layer(id string) (*storage.Layer, error)
	Image(id string) (*storage.Image, error)
	ImagesByTopLayer(id string) ([]*storage.Image, error)
	ImagesByDigest(d digest.Digest) ([]*storage.Image, error)
	Container(id string) (*storage.Container, error)
	ContainerByLayer(id string) (*storage.Container, error)
	ContainerDirectory(id string) (string, error)
	SetContainerDirectoryFile(id, file string, data []byte) error
	FromContainerDirectory(id, file string) ([]byte, error)
	ContainerRunDirectory(id string) (string, error)
	SetContainerRunDirectoryFile(id, file string, data []byte) error
	FromContainerRunDirectory(id, file string) ([]byte, error)
	ContainerParentOwners(id string) ([]int, []int, error)
	Lookup(name string) (string, error)
	Shutdown(force bool) (layers []string, err error)
	Version() ([][2]string, error)
	GetDigestLock(digest.Digest) (storage.Locker, error)
}

type mockStore struct {
	storage.Store
}

// NewMockStore return a mock store
func NewMockStore() MockStore {
	return &mockStore{}
}

func (m *mockStore) RunRoot() string {
	return ""
}

func (m *mockStore) GraphRoot() string {
	return ""
}

func (m *mockStore) GraphDriverName() string {
	return ""
}

func (m *mockStore) GraphOptions() []string {
	return nil
}

func (m *mockStore) UIDMap() []idtools.IDMap {
	return nil
}

func (m *mockStore) GIDMap() []idtools.IDMap {
	return nil
}

func (m *mockStore) GraphDriver() (drivers.Driver, error) {
	return nil, nil
}

func (m *mockStore) CreateLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions) (*storage.Layer, error) {
	return nil, nil
}

func (m *mockStore) PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *storage.LayerOptions, diff io.Reader) (*storage.Layer, int64, error) {
	return nil, 0, nil
}

func (m *mockStore) CreateImage(id string, names []string, layer, metadata string, options *storage.ImageOptions) (*storage.Image, error) {
	return nil, nil
}

func (m *mockStore) CreateContainer(id string, names []string, image, layer, metadata string, options *storage.ContainerOptions) (*storage.Container, error) {
	return &storage.Container{
		ID: "abcdedfg",
	}, nil
}

func (m *mockStore) Metadata(id string) (string, error) {
	return "", nil
}
func (m *mockStore) SetMetadata(id, metadata string) error {
	return nil
}

func (m *mockStore) Exists(id string) bool {
	return true
}

func (m *mockStore) Status() ([][2]string, error) {
	return [][2]string{}, nil
}

func (m *mockStore) Delete(id string) error {
	return nil
}

func (m *mockStore) DeleteLayer(id string) error {
	return nil
}
func (m *mockStore) DeleteImage(id string, commit bool) ([]string, error) {
	return nil, nil
}
func (m *mockStore) DeleteContainer(id string) error {
	return nil
}
func (m *mockStore) Wipe() error {
	return nil
}
func (m *mockStore) Mount(id, mountLabel string) (string, error) {
	return "", nil
}
func (m *mockStore) Unmount(id string, force bool) (bool, error) {
	return true, nil
}
func (m *mockStore) Changes(from, to string) ([]archive.Change, error) {
	return nil, nil
}
func (m *mockStore) DiffSize(from, to string) (int64, error) {
	return 0, nil
}
func (m *mockStore) Diff(from, to string, options *storage.DiffOptions) (io.ReadCloser, error) {
	r := strings.NewReader("isula-builder")
	return ioutil.NopCloser(r), nil
}

func (m *mockStore) ApplyDiff(to string, diff io.Reader) (int64, error) {
	return 0, nil
}

func (m *mockStore) LayersByCompressedDigest(d digest.Digest) ([]storage.Layer, error) {
	return nil, nil
}

func (m *mockStore) LayersByUncompressedDigest(d digest.Digest) ([]storage.Layer, error) {
	return nil, nil
}

func (m *mockStore) LayerSize(id string) (int64, error) {
	return 0, nil
}

func (m *mockStore) LayerParentOwners(id string) ([]int, []int, error) {
	return nil, nil, nil
}

func (m *mockStore) Layers() ([]storage.Layer, error) {
	return nil, nil
}

func (m *mockStore) Images() ([]storage.Image, error) {
	return nil, nil
}

func (m *mockStore) Containers() ([]storage.Container, error) {
	return nil, nil
}

func (m *mockStore) Names(id string) ([]string, error) {
	return nil, nil
}

func (m *mockStore) SetNames(id string, names []string) error {
	return nil
}

func (m *mockStore) ListImageBigData(id string) ([]string, error) {
	return nil, nil
}

func (m *mockStore) ImageBigData(id, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockStore) ImageBigDataSize(id, key string) (int64, error) {
	return 0, nil
}

func (m *mockStore) ImageBigDataDigest(id, key string) (digest.Digest, error) {
	return "", nil
}

func (m *mockStore) SetImageBigData(id, key string, data []byte, digestManifest func([]byte) (digest.Digest, error)) error {
	return nil
}

func (m *mockStore) ImageSize(id string) (int64, error) {
	return 0, nil
}

func (m *mockStore) ListContainerBigData(id string) ([]string, error) {
	return nil, nil
}

func (m *mockStore) ContainerBigData(id, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockStore) ContainerBigDataSize(id, key string) (int64, error) {
	return 0, nil
}

func (m *mockStore) ContainerBigDataDigest(id, key string) (digest.Digest, error) {
	return "", nil
}

func (m *mockStore) SetContainerBigData(id, key string, data []byte) error {
	return nil
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

func (m *mockStore) Image(id string) (*storage.Image, error) {
	return nil, nil
}

func (m *mockStore) ImagesByTopLayer(id string) ([]*storage.Image, error) {
	return nil, nil
}

func (m *mockStore) ImagesByDigest(d digest.Digest) ([]*storage.Image, error) {
	return nil, nil
}

func (m *mockStore) Container(id string) (*storage.Container, error) {
	return nil, nil
}

func (m *mockStore) ContainerByLayer(id string) (*storage.Container, error) {
	return nil, nil
}

func (m *mockStore) ContainerDirectory(id string) (string, error) {
	return "", nil
}

func (m *mockStore) SetContainerDirectoryFile(id, file string, data []byte) error {
	return nil
}

func (m *mockStore) FromContainerDirectory(id, file string) ([]byte, error) {
	return nil, nil
}

func (m *mockStore) ContainerRunDirectory(id string) (string, error) {
	return "", nil
}

func (m *mockStore) SetContainerRunDirectoryFile(id, file string, data []byte) error {
	return nil
}

func (m *mockStore) FromContainerRunDirectory(id, file string) ([]byte, error) {
	return nil, nil
}

func (m *mockStore) ContainerParentOwners(id string) ([]int, []int, error) {
	return nil, nil, nil
}

func (m *mockStore) Lookup(name string) (string, error) {
	return "", nil
}

func (m *mockStore) Shutdown(force bool) ([]string, error) {
	return nil, nil
}

func (m *mockStore) Version() ([][2]string, error) {
	return [][2]string{}, nil
}

func (m *mockStore) GetDigestLock(digest.Digest) (storage.Locker, error) {
	return nil, nil
}
