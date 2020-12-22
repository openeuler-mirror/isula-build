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
// Create: 2020-12-01
// Description: This file is used for manifest command.

package daemon

import (
	"context"
	"encoding/json"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/storage"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/builder/dockerfile/container"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const instancesData = "instancesdata"

type manifestList struct {
	docker    manifest.Schema2List
	instances map[digest.Digest]string
}

// ManifestCreate creates manifest list
func (b *Backend) ManifestCreate(ctx context.Context, req *pb.ManifestCreateRequest) (*pb.ManifestCreateResponse, error) {
	if !b.daemon.opts.Experimental {
		return &pb.ManifestCreateResponse{}, errors.New("please enable experimental to use manifest feature")
	}

	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
		"Manifest":     req.GetManifests(),
	}).Info("ManifestCreateRequest received")

	manifestName := req.GetManifestList()
	manifests := req.GetManifests()

	list := &manifestList{
		docker: manifest.Schema2List{
			SchemaVersion: container.SchemaVersion,
			MediaType:     manifest.DockerV2ListMediaType,
		},
		instances: make(map[digest.Digest]string, 0),
	}

	for _, imageSpec := range manifests {
		// add image to list
		if _, err := list.addImage(ctx, b.daemon.localStore, imageSpec); err != nil {
			return &pb.ManifestCreateResponse{}, err
		}
	}

	// expand list name
	_, imageName, err := dockerfile.CheckAndExpandTag(manifestName)
	if err != nil {
		return &pb.ManifestCreateResponse{}, err
	}
	// save list to image
	imageID, err := list.saveListToImage(b.daemon.localStore, "", imageName, list.docker.MediaType)

	return &pb.ManifestCreateResponse{
		ImageID: imageID,
	}, err
}

// ManifestAnnotate modifies and updates manifest list
func (b *Backend) ManifestAnnotate(ctx context.Context, req *pb.ManifestAnnotateRequest) (*gogotypes.Empty, error) {
	var emptyResp = &gogotypes.Empty{}

	if !b.daemon.opts.Experimental {
		return emptyResp, errors.New("please enable experimental to use manifest feature")
	}

	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
		"Manifest":     req.GetManifest(),
	}).Info("ManifestAnnotateRequest received")

	manifestName := req.GetManifestList()
	manifestImage := req.GetManifest()
	imageOS := req.GetOs()
	imageArch := req.GetArch()
	imageOSFeature := req.GetOsFeatures()
	imageVariant := req.GetVariant()

	// get list image
	_, listImage, err := image.FindImage(b.daemon.localStore, manifestName)
	if err != nil {
		return emptyResp, err
	}

	// load list from list image
	_, list, err := loadListFromImage(b.daemon.localStore, listImage.ID)
	if err != nil {
		return emptyResp, err
	}

	// add image to list, if image already exists, it will be substituted
	instanceDigest, err := list.addImage(ctx, b.daemon.localStore, manifestImage)
	if err != nil {
		return emptyResp, err
	}

	// modify image platform if user specifies
	for i := range list.docker.Manifests {
		if list.docker.Manifests[i].Digest == instanceDigest {
			if imageOS != "" {
				list.docker.Manifests[i].Platform.OS = imageOS
			}
			if imageArch != "" {
				list.docker.Manifests[i].Platform.Architecture = imageArch
			}
			if len(imageOSFeature) > 0 {
				list.docker.Manifests[i].Platform.OSFeatures = append([]string{}, imageOSFeature...)
			}
			if imageVariant != "" {
				list.docker.Manifests[i].Platform.Variant = imageVariant
			}
		}
	}

	// save list to image
	_, err = list.saveListToImage(b.daemon.localStore, listImage.ID, "", manifest.DockerV2ListMediaType)

	return emptyResp, err
}

// ManifestInspect inspects manifest list
func (b *Backend) ManifestInspect(ctx context.Context, req *pb.ManifestInspectRequest) (*pb.ManifestInspectResponse, error) {
	if !b.daemon.opts.Experimental {
		return &pb.ManifestInspectResponse{}, errors.New("please enable experimental to use manifest feature")
	}

	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
	}).Info("ManifestInspectRequest received")

	manifestName := req.GetManifestList()

	// get list image
	ref, _, err := image.FindImage(b.daemon.localStore, manifestName)
	if err != nil {
		return &pb.ManifestInspectResponse{}, err
	}

	// get image reference
	src, err := ref.NewImageSource(ctx, image.GetSystemContext())
	if err != nil {
		return &pb.ManifestInspectResponse{}, err
	}

	defer func() {
		if cErr := src.Close(); cErr != nil {
			logrus.Warnf("Image source closing error: %v", cErr)
		}
	}()

	// get image manifest
	manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
	if err != nil {
		return &pb.ManifestInspectResponse{}, err
	}

	// check whether image is a list image
	if !manifest.MIMETypeIsMultiImage(manifestType) {
		return &pb.ManifestInspectResponse{}, errors.Errorf("%v is not a manifest list", manifestName)
	}

	// return list image data
	return &pb.ManifestInspectResponse{
		Data: manifestBytes,
	}, nil
}

type instanceInfo struct {
	OS, Architecture string
	instanceDigest   *digest.Digest
	Size             int64
}

func (l *manifestList) addImage(ctx context.Context, store *store.Store, imageSpec string) (digest.Digest, error) {
	img, _, err := image.ResolveFromImage(&image.PrepareImageOptions{
		Ctx:           ctx,
		FromImage:     util.DefaultTransport + imageSpec,
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

func (l *manifestList) addInstance(instanceInfo instanceInfo, manifestType string) {
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

func (l *manifestList) removeInstance(instanceInfo instanceInfo) {
	newDockerManifests := make([]manifest.Schema2ManifestDescriptor, 0, len(l.docker.Manifests))

	for i := range l.docker.Manifests {
		if l.docker.Manifests[i].Digest != *instanceInfo.instanceDigest {
			newDockerManifests = append(newDockerManifests, l.docker.Manifests[i])
		}
	}

	l.docker.Manifests = newDockerManifests
}

func (l *manifestList) saveListToImage(store *store.Store, imageID, name string, mimeType string) (string, error) {
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
			logrus.Errorf("delete image %v as save manifest list to image failed error", imageID)
		}
		return "", errors.Wrapf(err, "save manifest list to image %v error", imageID)
	}

	//marshal list instance information
	instancesBytes, err := json.Marshal(&l.instances)
	if err != nil {
		return "", errors.Wrap(err, "marshall list instances error")
	}
	// save list.instance information to image
	if err = store.SetImageBigData(imageID, instancesData, instancesBytes, nil); err != nil {
		if _, err2 := store.DeleteImage(img.ID, true); err2 != nil {
			logrus.Errorf("delete image %v as save manifest list instance to image failed error", imageID)
		}
		return "", errors.Wrapf(err, "save manifest list instance to image %v error", imageID)
	}

	return imageID, nil
}

func loadListFromImage(store *store.Store, image string) (string, manifestList, error) {
	list := manifestList{
		docker: manifest.Schema2List{
			SchemaVersion: container.SchemaVersion,
			MediaType:     manifest.DockerV2ListMediaType,
		},
	}

	// get list image
	img, err := store.Image(image)
	if err != nil {
		return "", list, errors.Wrapf(err, "get image %v for loading manifest list error", image)
	}

	// load list.docker
	manifestBytes, err := store.ImageBigData(img.ID, storage.ImageDigestManifestBigDataNamePrefix)
	if err != nil {
		return "", list, errors.Wrapf(err, "get image data for loading manifest list error")
	}
	if err = json.Unmarshal(manifestBytes, &list.docker); err != nil {
		return "", list, errors.Wrapf(err, "parse image data to manifest list error")
	}

	// load list.instance
	instancesBytes, err := store.ImageBigData(img.ID, instancesData)
	if err != nil {
		return img.ID, list, errors.Wrapf(err, "get instance data for loading instance list error")
	}
	if err = json.Unmarshal(instancesBytes, &list.instances); err != nil {
		return img.ID, list, errors.Wrapf(err, "parse instance data to instance list error")
	}

	return img.ID, list, err
}
