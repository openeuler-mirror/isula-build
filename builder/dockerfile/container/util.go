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
// Description: container transport related common functions

package container

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/storage/pkg/archive"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	mimetypes "isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
)

func countDockerImageEmptyLayers(image docker.Image) int {
	cnt := 0
	for _, history := range image.History {
		if !history.EmptyLayer {
			cnt++
		}
	}

	return cnt
}

// Get the media type which will attach to the image layer
func getImageLayerMIMEType(layerCompression archive.Compression) (string, error) {
	dmediaType := mimetypes.DockerV2SchemaLayerMediaTypeUncompressed
	if layerCompression != archive.Uncompressed {
		switch layerCompression {
		case archive.Gzip:
			dmediaType = mimetypes.DockerV2Schema2LayerMediaType
		case archive.Bzip2:
			return "", errors.New("media type for bzip2-compressed layers is not support")
		case archive.Xz:
			return "", errors.New("media type for xz-compressed layers is not support")
		case archive.Zstd:
			return "", errors.New("media type for zstd-compressed layers is not support")
		default:
			return "", errors.New("unknown compression format")
		}
	}

	return dmediaType, nil
}

func encodeConfigsAndManifests(dimage docker.Image, dmanifest docker.Manifest, manifestType string) ([]byte, []byte, error) {
	// encode the config
	dimagebytes, err := json.Marshal(&dimage)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error encoding %#v as json", dimage)
	}

	// add the configuration blob to the manifest and encode it
	dmanifest.Config.Digest = digest.Canonical.FromBytes(dimagebytes)
	dmanifest.Config.Size = int64(len(dimagebytes))
	dmanifest.Config.MediaType = mimetypes.DockerV2Schema2ConfigMediaType
	dmanifestbytes, err := json.Marshal(&dmanifest)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error encoding %#v as json", dmanifest)
	}

	// decide which manifest and configuration blobs we'll actually output
	var config, manifest []byte
	switch manifestType {
	case mimetypes.MediaTypeImageManifest:
	case mimetypes.DockerV2Schema2MediaType:
		config, manifest = dimagebytes, dmanifestbytes
	default:
		return nil, nil, errors.Errorf("unsupported manifest type: %s", manifestType)
	}

	return config, manifest, nil
}

func appendHistory(dimage *docker.Image, historys []v1.History) {
	for _, history := range historys {
		var created *time.Time
		created = &time.Time{}
		if history.Created != nil {
			// here created time need a copy of history.Created
			copiedTimestamp := *history.Created
			created = &copiedTimestamp
		}

		dnews := docker.History{
			Created:    *created,
			CreatedBy:  history.CreatedBy,
			Author:     history.Author,
			Comment:    history.Comment,
			EmptyLayer: history.EmptyLayer,
		}
		dimage.History = append(dimage.History, dnews)
	}
}

// ChMTimeDir changes the access and modification times of the directory
func ChMTimeDir(dir string, buildTime time.Time) error {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(dir, entry.Name())
		if err := chMTime(srcPath, buildTime); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := ChMTimeDir(srcPath, buildTime); err != nil {
				return err
			}
		}
	}
	return nil
}

// chMTime changes the access and modification time of the files
func chMTime(path string, buildTime time.Time) error {
	if _, err := os.Lstat(path); err != nil {
		logrus.Warnf("Path %s is not exist", path)
		return nil
	}

	times := []unix.Timespec{
		unix.NsecToTimespec(buildTime.UnixNano()),
		unix.NsecToTimespec(buildTime.UnixNano()),
	}
	// update the atime and mtime on `path` without following the symlink
	return unix.UtimesNanoAt(unix.AT_FDCWD, path, times, unix.AT_SYMLINK_NOFOLLOW)
}
