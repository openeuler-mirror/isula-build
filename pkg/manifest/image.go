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
// Description: This file defines manifest image reference and manifest image source.

package manifest

import (
	"context"
	"io"

	"github.com/containers/image/v5/manifest"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
)

// manifestImageReference is image ImageReference implementation which used for manifest
type manifestImageReference struct {
	types.ImageReference                        // top image reference
	references           []types.ImageReference // instance images references
}

// manifestImageSource is ImageSource implementation which used for manifest
type manifestImageSource struct {
	types.ImageSource                                               // top image source
	imageSourceByInstanceDigest map[digest.Digest]types.ImageSource // mapping of instance digest to image source
	imageSourceByBlobDigest     map[digest.Digest]types.ImageSource // mapping of blob digest to image source
}

// Reference is used for manifest exporter getting image reference
func Reference(store *store.Store, manifestName string) (types.ImageReference, error) {
	// get list image
	_, listImage, err := image.FindImage(store, manifestName)
	if err != nil {
		logrus.Errorf("Manifest find image err: %v", err)
		return nil, err
	}

	// load list from list image
	list, err := LoadListFromImage(store, listImage.ID)
	if err != nil {
		logrus.Errorf("Manifest load list from image err: %v", err)
		return nil, err
	}

	// get list image reference
	sr, err := is.Transport.ParseStoreReference(store, listImage.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "parse image reference from image %v error", listImage.ID)
	}

	// get instances references
	references := make([]types.ImageReference, 0, len(list.instances))
	for _, instance := range list.instances {
		ref, err := alltransports.ParseImageName(instance)
		if err != nil {
			return nil, errors.Wrapf(err, "parse image reference from instance %v error", instance)
		}
		references = append(references, ref)
	}

	return &manifestImageReference{
		ImageReference: sr,
		references:     references,
	}, nil
}

// NewImageSource returns a types.ImageSource for this reference
func (s *manifestImageReference) NewImageSource(ctx context.Context, sys *types.SystemContext) (iss types.ImageSource, err error) {
	// get top image source
	src, err := s.ImageReference.NewImageSource(ctx, sys)
	if err != nil {
		return nil, errors.Wrapf(err, "new image source by image reference %v error", transports.ImageName(s.ImageReference))
	}

	defer func() {
		if err != nil {
			if iss != nil {
				if cErr := iss.Close(); cErr != nil {
					logrus.Errorf("Close image sources error: %v", cErr)
				}
			} else if src != nil {
				if cErr := src.Close(); cErr != nil {
					logrus.Errorf("Close top image source error: %v", cErr)
				}
			}
		}
	}()

	manifestIs := &manifestImageSource{
		ImageSource:                 src,
		imageSourceByInstanceDigest: make(map[digest.Digest]types.ImageSource),
		imageSourceByBlobDigest:     make(map[digest.Digest]types.ImageSource),
	}
	iss = manifestIs

	// get top image manifest digest
	manifestBytes, _, err := src.GetManifest(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get manifest bytes by image reference %v error", transports.ImageName(s.ImageReference))
	}
	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "get manifest digest by image reference %v error", transports.ImageName(s.ImageReference))
	}

	// write top image mapping info to image source map
	manifestIs.imageSourceByInstanceDigest[""] = src
	manifestIs.imageSourceByInstanceDigest[manifestDigest] = src

	// get instance images manifest digest and blob digest, and write mapping info to image source map
	visited := make(map[types.ImageReference]struct{})
	for _, ref := range s.references {
		// skip if reference already be visited
		if _, visited := visited[ref]; visited {
			continue
		}
		visited[ref] = struct{}{}

		// get instance image source
		src, err := ref.NewImageSource(ctx, sys)
		if err != nil {
			return nil, errors.Wrapf(err, "new image source by image reference %v error", transports.ImageName(ref))
		}

		// get instance image manifest digest and write to map
		manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "get manifest bytes by image reference %v error", transports.ImageName(ref))
		}
		manifestDigest, err := manifest.Digest(manifestBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "get manifest digest by image reference %v error", transports.ImageName(ref))
		}

		manifestIs.imageSourceByInstanceDigest[manifestDigest] = src

		manifest, err := manifest.FromBlob(manifestBytes, manifestType)
		if err != nil {
			return nil, errors.Wrapf(err, "parse manifest %v error", string(manifestBytes))
		}

		// get instance image config info
		config := manifest.ConfigInfo()
		if config.Digest != "" {
			manifestIs.imageSourceByBlobDigest[config.Digest] = src
		}

		// get instance image layer info
		layers := manifest.LayerInfos()
		for _, layer := range layers {
			manifestIs.imageSourceByBlobDigest[layer.Digest] = src
		}
	}

	return iss, nil
}

func (s *manifestImageSource) getImgSrc(instanceDigest *digest.Digest) types.ImageSource {
	var src types.ImageSource
	if instanceDigest == nil {
		src = s.imageSourceByInstanceDigest[""]
	} else {
		src = s.imageSourceByInstanceDigest[*instanceDigest]
	}

	return src
}

// GetManifest returns the image's manifest along with its MIME type
func (s *manifestImageSource) GetManifest(ctx context.Context, instanceDigest *digest.Digest) ([]byte, string, error) {
	if src := s.getImgSrc(instanceDigest); src != nil {
		return src.GetManifest(ctx, nil)
	}

	return nil, "", errors.Errorf("get manifest for instance digest %v failed", *instanceDigest)
}

// GetSignatures returns the image's signatures
func (s *manifestImageSource) GetSignatures(ctx context.Context, instanceDigest *digest.Digest) ([][]byte, error) {
	if src := s.getImgSrc(instanceDigest); src != nil {
		return src.GetSignatures(ctx, nil)
	}

	return nil, errors.Errorf("get signatures for instance digest %v failed", *instanceDigest)
}

// LayerInfosForCopy returns either nil (meaning the values in the manifest are fine), or updated values for the layer
// blobsums that are listed in the image's manifest
func (s *manifestImageSource) LayerInfosForCopy(ctx context.Context, instanceDigest *digest.Digest) ([]types.BlobInfo, error) {
	src := s.getImgSrc(instanceDigest)
	if src == nil {
		logrus.Errorf("Get image source for instance digest %v failed", *instanceDigest)
		return nil, errors.Errorf("get image source for instance digest %v failed", *instanceDigest)
	}

	blobInfos, err := src.LayerInfosForCopy(ctx, nil)
	if err != nil {
		logrus.Errorf("Get layer infos for copy from instance digest %v error: %v", *instanceDigest, err)
		return nil, errors.Wrapf(err, "get layer infos for copy from instance digest %v error", *instanceDigest)
	}

	for _, blobInfo := range blobInfos {
		s.imageSourceByBlobDigest[blobInfo.Digest] = src
	}

	return blobInfos, nil
}

// GetBlob returns a stream for the specified blob, and the blob’s size
func (s *manifestImageSource) GetBlob(ctx context.Context, blob types.BlobInfo, bic types.BlobInfoCache) (io.ReadCloser, int64, error) {
	src, ok := s.imageSourceByBlobDigest[blob.Digest]
	if !ok {
		return nil, -1, errors.Errorf("get image source by blob digest %v failed", blob.Digest)
	}

	return src.GetBlob(ctx, blob, bic)
}

// HasThreadSafeGetBlob indicates whether GetBlob can be executed concurrently
func (s *manifestImageSource) HasThreadSafeGetBlob() bool {
	for _, sourceInstance := range s.imageSourceByInstanceDigest {
		if !sourceInstance.HasThreadSafeGetBlob() {
			return false
		}
	}

	return true
}

// Close removes resources associated with an initialized ImageSource, if any
func (s *manifestImageSource) Close() error {
	var retErr error
	closed := make(map[types.ImageSource]struct{})

	for _, imageSource := range s.imageSourceByInstanceDigest {
		// skip if image source is already closed
		if _, isClosed := closed[imageSource]; isClosed {
			continue
		}
		closed[imageSource] = struct{}{}

		if err := imageSource.Close(); err != nil {
			logrus.Errorf("Close imageSource %v error: %v", imageSource, err)
			retErr = err
		}
	}

	return retErr
}
