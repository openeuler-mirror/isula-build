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
// Description: container transport related functions

// Package container is used for container transport
package container

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/image"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	mimetypes "isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

// Reference is the struct of a commit container's metadata
type Reference struct {
	store                 *store.Store
	compression           archive.Compression
	name                  reference.Named
	containerID           string
	layerID               string
	dconfig               []byte
	created               time.Time
	createdBy             string
	historyComment        string
	annotations           map[string]string
	preferredManifestType string
	exporting             bool
	fixed                 bool
	emptyLayer            bool
	parent                string
	preEmptyLayers        []v1.History
	postEmptyLayers       []v1.History
	tarPath               func(path string) (io.ReadCloser, error)
}

// ReferenceMetadata is the struct of a commit container's reference metadata
type ReferenceMetadata struct {
	BuildTime             *time.Time
	CreatedBy             string
	HistoryComment        string
	Compression           archive.Compression
	Annotations           map[string]string
	Parent                string
	PreferredManifestType string
	Dconfig               []byte
	Name                  reference.Named
	ContainerID           string
	LayerID               string
	EmptyLayer            bool
	PreEmptyLayers        []v1.History
	PostEmptyLayers       []v1.History
	TarPath               func(path string) (io.ReadCloser, error)
}

// NewContainerReference return a container reference
func NewContainerReference(store *store.Store, metadata *ReferenceMetadata, exporting bool) Reference {
	ref := Reference{
		store:                 store,
		exporting:             exporting,
		name:                  metadata.Name,
		compression:           metadata.Compression,
		containerID:           metadata.ContainerID,
		layerID:               metadata.LayerID,
		dconfig:               metadata.Dconfig,
		created:               time.Now().UTC(),
		createdBy:             metadata.CreatedBy,
		historyComment:        metadata.HistoryComment,
		annotations:           metadata.Annotations,
		preferredManifestType: mimetypes.DockerV2Schema2MediaType,
		emptyLayer:            metadata.EmptyLayer,
		tarPath:               metadata.TarPath,
		parent:                metadata.Parent,
		preEmptyLayers:        metadata.PreEmptyLayers,
		postEmptyLayers:       metadata.PostEmptyLayers,
	}
	if metadata.BuildTime != nil {
		ref.created = *metadata.BuildTime
		ref.fixed = true
	}
	return ref
}

// NewImage returns an image closer
func (ref *Reference) NewImage(ctx context.Context, sc *types.SystemContext) (types.ImageCloser, error) {
	src, err := ref.NewImageSource(ctx, sc)
	if err != nil {
		return nil, err
	}
	return image.FromSource(ctx, sc, src)
}

// NewImageSource return a types.ImageSource instance
func (ref *Reference) NewImageSource(ctx context.Context, sc *types.SystemContext) (src types.ImageSource, err error) {
	// 1. prepare all layers for this refeference
	layers, err := ref.getContainerLayers()
	if err != nil {
		return nil, errors.Wrapf(err, "get build container layers failed")
	}

	buildDirValue := ctx.Value(util.BuildDirKey(util.BuildDir))
	buildDir, ok := buildDirValue.(string)
	if !ok {
		return nil, errors.Errorf("buildDirValue %+v assert to string failed", buildDirValue)
	}
	blobDir := filepath.Join(buildDir, "blob")
	if err = os.MkdirAll(blobDir, constant.DefaultRootDirMode); err != nil {
		return nil, err
	}

	// 2. new a copy of the configurations and manifest
	dimage, dmanifest, err := ref.createConfigsAndManifests()
	if err != nil {
		return nil, err
	}

	// 3. analyze each layer and compute its digests, both compressed (if requested) and uncompressed
	for _, layerID := range layers {
		if ref.emptyLayer && layerID == ref.layerID {
			continue
		}
		diffID, dlayerDescriptor, err2 := ref.analyzeLayer(layerID, blobDir)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "analyze layer %q failed", layerID)
		}
		dimage.RootFS.DiffIDs = append(dimage.RootFS.DiffIDs, diffID)
		dmanifest.Layers = append(dmanifest.Layers, dlayerDescriptor)
	}

	// 4. build history in the image
	ref.appendHistory(&dimage)

	// 5. ensure that not just create a mismatch between non-empty layers in the history and the number of diffIDs
	cnt := countDockerImageEmptyLayers(dimage)
	if len(dimage.RootFS.DiffIDs) != cnt {
		return nil, errors.Errorf("history lists %d non-empty layers, but have %d layers on disk", cnt, len(dimage.RootFS.DiffIDs))
	}

	// 6. new a containerImageSource instance base above information
	src, err = ref.newImageSource(blobDir, dimage, dmanifest)
	if err != nil {
		return nil, errors.Wrap(err, "new image source failed")
	}
	return src, nil
}

func (ref *Reference) newImageSource(path string, dimage docker.Image, dmanifest docker.Manifest) (*containerImageSource, error) {
	manifestType := ref.preferredManifestType
	if manifestType != mimetypes.MediaTypeImageManifest && manifestType != mimetypes.DockerV2Schema2MediaType {
		return nil, errors.Errorf("the manifest type: %q is not support yet", manifestType)
	}

	config, manifest, err := encodeConfigsAndManifests(dimage, dmanifest, manifestType)
	if err != nil {
		return nil, err
	}

	src := &containerImageSource{
		path:         path,
		ref:          ref,
		store:        ref.store,
		containerID:  ref.containerID,
		layerID:      ref.layerID,
		compression:  ref.compression,
		config:       config,
		configDigest: digest.Canonical.FromBytes(config),
		manifest:     manifest,
		manifestType: manifestType,
		exporting:    ref.exporting,
	}

	return src, nil
}

// NewImageDestination return a types.ImageDestination instance
func (ref *Reference) NewImageDestination(ctx context.Context, sc *types.SystemContext) (types.ImageDestination, error) {
	return nil, errors.Errorf("can't write to a container")
}

// DockerReference return container's reference
func (ref *Reference) DockerReference() reference.Named {
	return ref.name
}

// StringWithinTransport return transport name
func (ref *Reference) StringWithinTransport() string {
	return "container"
}

// DeleteImage delete an image
func (ref *Reference) DeleteImage(context.Context, *types.SystemContext) error {
	return nil
}

// PolicyConfigurationIdentity check policy config
func (ref *Reference) PolicyConfigurationIdentity() string {
	return ""
}

// PolicyConfigurationNamespaces return policy config namespace
func (ref *Reference) PolicyConfigurationNamespaces() []string {
	return nil
}

// Transport return the transport
func (ref *Reference) Transport() types.ImageTransport {
	return is.Transport
}
