// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2021-01-01
// Description: This file defines manifest list and related functions.

package manifest

import (
	"context"
	"encoding/json"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/storage"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile/container"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
)

const instancesData = "instancesdata"

// List is the list with all manifest information
type List struct {
	docker    manifest.Schema2List
	instances map[digest.Digest]string
}

type instanceInfo struct {
	OS, Architecture string
	instanceDigest   *digest.Digest
	Size             int64
}

// NewManifestList returns a new manifest list
func NewManifestList() *List {
	return &List{
		docker: manifest.Schema2List{
			SchemaVersion: container.SchemaVersion,
			MediaType:     manifest.DockerV2ListMediaType,
		},
		instances: make(map[digest.Digest]string, 0),
	}
}

// AddImage adds image to manifest
func (l *List) AddImage(ctx context.Context, store *store.Store, imageSpec string) (digest.Digest, error) {
	img, _, err := image.ResolveFromImage(&image.PrepareImageOptions{
		Ctx:           ctx,
		FromImage:     exporter.FormatTransport(exporter.DockerTransport, imageSpec),
		SystemContext: image.GetSystemContext(),
		Store:         store,
	})
	if err != nil {
		return "", err
	}

	var instance instanceInfo
	// get image OS and architecture
	config, err := img.OCIConfig(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "get oci config from image %v error", imageSpec)
	}
	instance.OS = config.OS
	instance.Architecture = config.Architecture

	// get image manifest digest and size
	manifestBytes, manifestType, err := img.Manifest(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "get manifest from image %v error", imageSpec)
	}
	if manifest.MIMETypeIsMultiImage(manifestType) {
		return "", errors.Errorf("%v is a manifest list", imageSpec)
	}
	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		return "", errors.Wrapf(err, "compute digest of manifest from image %v error", imageSpec)
	}
	instance.instanceDigest = &manifestDigest
	instance.Size = int64(len(manifestBytes))

	// add image information to list
	l.addInstance(instance, manifestType)

	// update list instances
	if _, ok := l.instances[*instance.instanceDigest]; !ok {
		l.instances[*instance.instanceDigest] = transports.ImageName(img.Reference())
	}

	return manifestDigest, nil
}

// SaveListToImage saves list information to an image
func (l *List) SaveListToImage(store *store.Store, imageID, name string) (string, error) {
	// create an image to store list information
	img, err := store.CreateImage(imageID, []string{name}, "", "", &storage.ImageOptions{})
	if err != nil && errors.Cause(err) != storage.ErrDuplicateID {
		return "", errors.Wrap(err, "create image to store manifest list error")
	}

	// err == nil means a new image is created, if not, means image already exists, just modify image data
	if err == nil {
		imageID = img.ID
	}

	// marshal list information
	manifestBytes, err := json.Marshal(&l.docker)
	if err != nil {
		return "", errors.Wrap(err, "marshall Docker manifest list error")
	}
	// save list.docker information to image
	if err = store.SetImageBigData(imageID, storage.ImageDigestManifestBigDataNamePrefix, manifestBytes, manifest.Digest); err != nil {
		if _, err2 := store.DeleteImage(img.ID, true); err2 != nil {
			logrus.Errorf("Delete image %v as save manifest list to image failed error", imageID)
		}
		return "", errors.Wrapf(err, "save manifest list to image %v error", imageID)
	}

	// marshal list instance information
	instancesBytes, err := json.Marshal(&l.instances)
	if err != nil {
		return "", errors.Wrap(err, "marshall list instances error")
	}
	// save list.instance information to image
	if err = store.SetImageBigData(imageID, instancesData, instancesBytes, nil); err != nil {
		if _, err2 := store.DeleteImage(img.ID, true); err2 != nil {
			logrus.Errorf("Delete image %v as save manifest list instance to image failed error", imageID)
		}
		return "", errors.Wrapf(err, "save manifest list instance to image %v error", imageID)
	}

	return imageID, nil
}

// UpdateImagePlatform updates image platform information in the list if user specifies
func (l *List) UpdateImagePlatform(req *pb.ManifestAnnotateRequest, instanceDigest digest.Digest) {
	imageOS := req.GetOs()
	imageArch := req.GetArch()
	imageOSFeature := req.GetOsFeatures()
	imageVariant := req.GetVariant()

	for i := range l.docker.Manifests {
		if l.docker.Manifests[i].Digest == instanceDigest {
			if imageOS != "" {
				l.docker.Manifests[i].Platform.OS = imageOS
			}
			if imageArch != "" {
				l.docker.Manifests[i].Platform.Architecture = imageArch
			}
			if len(imageOSFeature) > 0 {
				l.docker.Manifests[i].Platform.OSFeatures = append([]string{}, imageOSFeature...)
			}
			if imageVariant != "" {
				l.docker.Manifests[i].Platform.Variant = imageVariant
			}
		}
	}
}

// addInstance adds image instance to list
func (l *List) addInstance(instanceInfo instanceInfo, manifestType string) {
	// remove instance if it is already exists, as we want to substitute it
	l.removeInstance(instanceInfo)

	schema2platform := manifest.Schema2PlatformSpec{
		Architecture: instanceInfo.Architecture,
		OS:           instanceInfo.OS,
	}

	l.docker.Manifests = append(l.docker.Manifests, manifest.Schema2ManifestDescriptor{
		Schema2Descriptor: manifest.Schema2Descriptor{
			MediaType: manifestType,
			Size:      instanceInfo.Size,
			Digest:    *instanceInfo.instanceDigest,
		},
		Platform: schema2platform,
	})
}

// removeInstance removes image instance from list
func (l *List) removeInstance(instanceInfo instanceInfo) {
	newDockerManifests := make([]manifest.Schema2ManifestDescriptor, 0, len(l.docker.Manifests))

	for i := range l.docker.Manifests {
		if l.docker.Manifests[i].Digest != *instanceInfo.instanceDigest {
			newDockerManifests = append(newDockerManifests, l.docker.Manifests[i])
		}
	}

	l.docker.Manifests = newDockerManifests
}

// LoadListFromImage load list from the stored image
func LoadListFromImage(store *store.Store, imageID string) (*List, error) {
	list := NewManifestList()

	// load list.docker
	manifestBytes, err := store.ImageBigData(imageID, storage.ImageDigestManifestBigDataNamePrefix)
	if err != nil {
		return list, errors.Wrapf(err, "get image data for loading manifest list error")
	}
	if err = json.Unmarshal(manifestBytes, &list.docker); err != nil {
		return list, errors.Wrapf(err, "parse image data to manifest list error")
	}

	// load list.instance
	instancesBytes, err := store.ImageBigData(imageID, instancesData)
	if err != nil {
		return list, errors.Wrapf(err, "get instance data for loading instance list error")
	}
	if err = json.Unmarshal(instancesBytes, &list.instances); err != nil {
		return list, errors.Wrapf(err, "parse instance data to instance list error")
	}

	return list, err
}
