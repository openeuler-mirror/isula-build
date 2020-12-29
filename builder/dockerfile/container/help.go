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

package container

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	mimetypes "isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
)

const (
	// SchemaVersion is the image manifest schema
	SchemaVersion = 2
)

// create configs and manifests aimed to can edit without making unintended changes
func (ref *Reference) createConfigsAndManifests() (docker.Image, docker.Manifest, error) {
	// 1. image
	dimage := docker.Image{}
	if err := json.Unmarshal(ref.dconfig, &dimage); err != nil {
		return docker.Image{}, docker.Manifest{}, err
	}
	dimage.Parent = docker.ID(ref.parent)
	dimage.Container = ref.containerID
	if dimage.Config != nil {
		dimage.ContainerConfig = *dimage.Config
	}
	dimage.Created = ref.created
	dimage.RootFS = &docker.RootFS{}
	dimage.RootFS.Type = docker.TypeLayers
	dimage.RootFS.DiffIDs = []digest.Digest{}

	// 2. manifest
	dmanifest := docker.Manifest{
		Versioned: docker.Versioned{
			SchemaVersion: SchemaVersion,
			MediaType:     mimetypes.DockerV2Schema2MediaType,
		},
		Config: docker.Descriptor{
			MediaType: mimetypes.DockerV2Schema2ConfigMediaType,
		},
		Layers: []docker.Descriptor{},
	}

	return dimage, dmanifest, nil
}

func (ref *Reference) getContainerLayers() ([]string, error) {
	layers := make([]string, 0, 0)
	layerID := ref.layerID
	layer, err := ref.store.Layer(layerID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read layer %q", layerID)
	}

	// add each layer in the layers until reach the root layer
	for layer != nil {
		layers = append(layers, layerID)
		layerID = layer.Parent
		if layerID == "" {
			err = nil
			break
		}
		layer, err = ref.store.Layer(layerID)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read layer %q", layerID)
		}
	}

	// reverse layers
	i, j := 0, len(layers)-1
	for i < j {
		layers[i], layers[j] = layers[j], layers[i]
		i, j = i+1, j-1
	}

	return layers, err
}

func (ref *Reference) analyzeLayer(layerID, path string) (digest.Digest, docker.Descriptor, error) {
	layer, err := ref.store.Layer(layerID)
	if err != nil {
		return "", docker.Descriptor{}, errors.Wrapf(err, "unable to find the layer")
	}

	// if not exporting and not the top layer, reusing layers individually, include blobsum and diff IDs
	if !ref.exporting && layerID != ref.layerID {
		return ref.reuseLayer(layer)
	}

	return ref.saveLayerToStorage(path, layer)

}

func (ref *Reference) reuseLayer(layer *storage.Layer) (digest.Digest, docker.Descriptor, error) {
	if layer.UncompressedDigest == "" {
		return "", docker.Descriptor{}, errors.New("uncompressed digest is empty, unable to find the layer")
	}
	diffID := layer.UncompressedDigest
	dlayerDescriptor := docker.Descriptor{
		MediaType: mimetypes.DockerV2Schema2LayerMediaType,
		Digest:    layer.UncompressedDigest,
		Size:      layer.UncompressedSize,
	}

	return diffID, dlayerDescriptor, nil
}

func (ref *Reference) prepareTarStream(layer *storage.Layer) (io.ReadCloser, error) {
	var (
		err error
		rc  io.ReadCloser
	)
	// when layer mount point is not empty, it's a upper layer, need to change
	// the modify time when build image with fixed time
	if ref.fixed && layer.MountPoint != "" {
		diff := filepath.Join(filepath.Dir(layer.MountPoint), "diff")
		if err = ChMTimeDir(diff, ref.created); err != nil {
			return nil, errors.Wrapf(err, "change time of layer's diff failed")
		}
	}
	noCompression := archive.Uncompressed
	diffOptions := &storage.DiffOptions{
		Compression: &noCompression,
	}
	if rc, err = ref.store.Diff("", layer.ID, diffOptions); err != nil {
		return nil, err
	}

	return rc, nil
}

func (ref *Reference) saveLayerToStorage(path string, layer *storage.Layer) (diffID digest.Digest, des docker.Descriptor, err error) {
	dmediaType, err := getImageLayerMIMEType(ref.compression)
	if err != nil {
		return "", des, err
	}

	rc, err := ref.prepareTarStream(layer)
	if err != nil {
		return "", des, err
	}
	defer func() {
		if err2 := rc.Close(); err2 != nil {
			logrus.Warnf("Close rootfs stream failed: %s", err2.Error())
		}
	}()

	filename := filepath.Join(path, "layer")
	layerFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, constant.DefaultRootFileMode)
	if err != nil {
		return "", des, errors.Wrapf(err, "error opening file: %s", filename)
	}
	defer func() {
		if err2 := layerFile.Close(); err2 != nil {
			logrus.Warnf("layer file close failed: %s", err2.Error())
		}
	}()

	diffID, des, err = ref.storeLayer(path, layerFile, rc)
	if err != nil {
		return "", des, nil
	}
	des.MediaType = dmediaType

	return diffID, des, nil
}

func (ref *Reference) storeLayer(path string, layerFile *os.File, rc io.ReadCloser) (diffID digest.Digest, des docker.Descriptor, err error) {
	srcHasher := digest.Canonical.Digester()
	reader := io.TeeReader(rc, srcHasher.Hash())
	destHasher := digest.Canonical.Digester()
	counter := ioutils.NewWriteCounter(layerFile)
	multiWriter := io.MultiWriter(counter, destHasher.Hash())
	writer, err := archive.CompressStream(multiWriter, ref.compression)
	if err != nil {
		return "", des, errors.Wrapf(err, "error compressing")
	}

	size, err := io.Copy(writer, reader)
	if err != nil {
		err = errors.Wrap(err, "error storing to file, copy failed")
		if werr := writer.Close(); werr != nil {
			err = errors.Wrap(err, werr.Error())
		}
		return "", des, err
	}
	if err2 := writer.Close(); err2 != nil {
		return "", des, errors.Wrap(err2, "writer close failed after copy")
	}
	if ref.compression != archive.Uncompressed {
		size = counter.Count
	}
	if ref.compression == archive.Uncompressed && size != counter.Count {
		return "", des, errors.Errorf("error storing file: inconsistent layer size (copied %d, wrote %d)", size, counter.Count)
	}
	// rename the layer so that we can more easily find it by digest later
	finalBlobName := filepath.Join(path, destHasher.Digest().String())
	if err = os.Rename(filepath.Join(path, "layer"), finalBlobName); err != nil {
		return "", des, errors.Wrapf(err, "error storing to file while renaming %q to %q", filepath.Join(path, "layer"), finalBlobName)
	}
	des = docker.Descriptor{
		Digest: destHasher.Digest(),
		Size:   size,
	}
	diffID = srcHasher.Digest()

	return diffID, des, nil
}

func (ref *Reference) appendHistory(dimage *docker.Image) {
	appendHistory(dimage, ref.preEmptyLayers)
	dnews := docker.History{
		Created:    ref.created,
		CreatedBy:  ref.createdBy,
		Author:     dimage.Author,
		Comment:    ref.historyComment,
		EmptyLayer: ref.emptyLayer,
	}
	dimage.History = append(dimage.History, dnews)
	appendHistory(dimage, ref.postEmptyLayers)
}
