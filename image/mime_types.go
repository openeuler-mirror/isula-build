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
// Description: mime related constants

package image

import (
	"github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// MediaTypeImageManifest specifies the media type for an image manifest
	// which value is "application/vnd.oci.image.manifest.v1+json"
	MediaTypeImageManifest = imgspecv1.MediaTypeImageManifest

	// DockerV2Schema2MediaType MIME type represents Docker manifest schema 2
	// which value is "application/vnd.docker.distribution.manifest.v2+json"
	DockerV2Schema2MediaType = manifest.DockerV2Schema2MediaType

	// DockerV2Schema2LayerMediaType is a MIME type for Docker image schema 2 layers
	// which value is "application/vnd.docker.image.rootfs.diff.tar.gzip"
	DockerV2Schema2LayerMediaType = manifest.DockerV2Schema2LayerMediaType

	// DockerV2Schema2ConfigMediaType is the MIME type used for schema 2 config blobs
	// which value is "application/vnd.docker.container.image.v1+json"
	DockerV2Schema2ConfigMediaType = manifest.DockerV2Schema2ConfigMediaType

	// DockerV2SchemaLayerMediaTypeUncompressed is the mediaType used for uncompressed layers
	// which value is "application/vnd.docker.image.rootfs.diff.tar"
	DockerV2SchemaLayerMediaTypeUncompressed = manifest.DockerV2SchemaLayerMediaTypeUncompressed

	// DockerV2Schema1SignedMediaType is the mediaType used for JWS signature
	// which value is "application/vnd.docker.distribution.manifest.v1+prettyjw"
	DockerV2Schema1SignedMediaType = manifest.DockerV2Schema1SignedMediaType
)
