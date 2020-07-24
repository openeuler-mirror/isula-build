// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-03-20
// Description: structure extracted from Docker

//
//  Since following code part extracted from Docker, their copyright
//  is retained here....
//
//  Copyright (C) Copyright 2013-2020 Docker, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//  http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

// Package docker saves the structure extracted from Docker
package docker

//
//  Types extracted from Docker
//  The following part extracted from:
//  https://github.com/moby/moby
//  commit cc0dfb6e7b22ad120c60a9ce770ea15415767cf9
//  https://github.com/docker/go-connections
//  commit 7dc0a2d6ddce55257ea8851e23b4fb9ef44fd4a0
//  https://github.com/docker/distribution
//  commit 742aab907b54a367e1ac7033fb9fe73b0e7344f5
//

import (
	"time"

	"github.com/containers/image/v5/pkg/strslice"
	digest "github.com/opencontainers/go-digest"
)

// TypeLayers is used for RootFS.Type for filesystems organized into layers.
// github.com/moby/moby/image/rootfs.go
const TypeLayers = "layers"

// RootFS describes images root filesystem
// This is currently a placeholder that only supports layers. In the future
// this can be made into an interface that supports different implementations.
// github.com/moby/moby/image/rootfs.go
type RootFS struct {
	DiffIDs []digest.Digest `json:"diff_ids,omitempty"`
	Type    string          `json:"type"`
}

// History stores build commands that were used to create an image
// github.com/moby/moby/image/image.go
type History struct {
	// Created is the timestamp at which the image was created
	Created time.Time `json:"created"`
	// Author is the name of the author that was specified when committing the image
	Author string `json:"author,omitempty"`
	// CreatedBy keeps the Dockerfile command used while building the image
	CreatedBy string `json:"created_by,omitempty"`
	// Comment is the commit message that was set when committing the image
	Comment string `json:"comment,omitempty"`
	// EmptyLayer is set to true if this history item did not generate a
	// layer. Otherwise, the history item is associated with the next
	// layer in the RootFS section.
	EmptyLayer bool `json:"empty_layer,omitempty"`
}

// ID is the content-addressable ID of an image.
// github.com/moby/moby/image/image.go
type ID digest.Digest

// HealthConfig holds configuration settings for the HEALTHCHECK feature.
// github.com/moby/moby/api/types/container/config.go
type HealthConfig struct {
	// Test is the test to perform to check that the container is healthy.
	// An empty slice means to inherit the default.
	// The options are:
	// {} : inherit healthcheck
	// {"NONE"} : disable healthcheck
	// {"CMD", args...} : exec arguments directly
	// {"CMD-SHELL", command} : run command with system's default shell
	Test []string `json:",omitempty"`

	// Zero means to inherit. Durations are expressed as integer nanoseconds.
	Interval    time.Duration `json:",omitempty"` // Interval is the time to wait between checks.
	Timeout     time.Duration `json:",omitempty"` // Timeout is the time to wait before considering the check to have hung.
	StartPeriod time.Duration `json:",omitempty"` // Time to wait after the container starts before running the first check.

	// Retries is the number of consecutive failures needed to consider a container as unhealthy.
	// Zero means inherit.
	Retries int `json:",omitempty"`
}

// PortSet is a collection of structs indexed by Port
// github.com/docker/go-connections/nat/nat.go
type PortSet map[Port]struct{}

// Port is a string containing port number and protocol in the format "80/tcp"
// github.com/docker/go-connections/nat/nat.go
type Port string

// Config contains the configuration data about a container.
// It should hold only portable information about the container.
// Here, "portable" means "independent from the host we are running on".
// Non-portable information *should* appear in HostConfig.
// All fields added to this struct must be marked `omitempty` to keep getting
// predictable hashes from the old `v1Compatibility` configuration.
// github.com/moby/moby/api/types/container/config.go
type Config struct {
	Healthcheck  *HealthConfig       `json:",omitempty"` // Healthcheck describes how to check the container is healthy
	StopTimeout  *int                `json:",omitempty"` // Timeout (in seconds) to stop a container
	Shell        strslice.StrSlice   `json:",omitempty"` // Shell for shell-form of RUN, CMD, ENTRYPOINT
	Cmd          strslice.StrSlice   // Command to run when starting the container
	Entrypoint   strslice.StrSlice   // Entrypoint to run when starting the container
	Hostname     string              // Hostname
	User         string              // User that will run the command(s) inside the container, also support user:group
	WorkingDir   string              // Current directory (PWD) in the command will be launched
	StopSignal   string              `json:",omitempty"` // Signal to stop a container
	ExposedPorts PortSet             `json:",omitempty"` // List of exposed ports
	Env          []string            // List of environment variable to set in the container
	OnBuild      []string            // ONBUILD metadata that were defined on the image Dockerfile
	Volumes      map[string]struct{} // List of volumes (mounts) used for the container
	Labels       map[string]string   // List of labels set to this container
}

// V1Image stores the V1 image configuration.
// github.com/moby/moby/image/image.go
type V1Image struct {
	// Config is the configuration of the container received from the client
	Config *Config `json:"config,omitempty"`
	// ContainerConfig is the configuration of the container that is committed into the image
	ContainerConfig Config `json:"container_config,omitempty"`
	// ID is a unique 64 character identifier of the image
	ID string `json:"id,omitempty"`
	// Parent is the ID of the parent image
	Parent string `json:"parent,omitempty"`
	// Comment is the commit message that was set when committing the image
	Comment string `json:"comment,omitempty"`
	// DockerVersion specifies the version of Docker that was used to build the image
	DockerVersion string `json:"docker_version,omitempty"`
	// Author is the name of the author that was specified when committing the image
	Author string `json:"author,omitempty"`
	// Architecture is the hardware that the image is build and runs on
	Architecture string `json:"architecture,omitempty"`
	// OS is the operating system used to build and run the image
	OS string `json:"os,omitempty"`
	// Container is the id of the container used to commit
	Container string `json:"container,omitempty"`
	// Size is the total size of the image including all layers it is composed of
	Size int64 `json:",omitempty"`
	// Created is the timestamp at which the image was created
	Created time.Time `json:"created"`
}

// Image stores the image configuration
// github.com/moby/moby/image/image.go
type Image struct {
	RootFS  *RootFS   `json:"rootfs,omitempty"`
	History []History `json:"history,omitempty"`
	Parent  ID        `json:"parent,omitempty"` // nolint:govet
	V1Image
}

// Versioned provides a struct with the manifest schemaVersion and mediaType.
// Incoming content with unknown schema version can be decoded against this
// struct to check the version.
// github.com/docker/distribution/manifest/versioned.go
type Versioned struct {
	// SchemaVersion is the image manifest schema that this image follows
	SchemaVersion int `json:"schemaVersion"`

	// MediaType is the media type of this schema.
	MediaType string `json:"mediaType,omitempty"`
}

// Descriptor describes targeted content. Used in conjunction with a blob
// store, a descriptor can be used to fetch, store and target any kind of
// blob. The struct also describes the wire protocol format. Fields should
// only be added but never changed.
// github.com/docker/distribution/blobs.go
type Descriptor struct {
	// MediaType describe the type of the content. All text based formats are
	// encoded as utf-8.
	MediaType string `json:"mediaType,omitempty"`

	// Size in bytes of content.
	Size int64 `json:"size,omitempty"`

	// Digest uniquely identifies the content. A byte stream can be verified
	// against against this digest.
	Digest digest.Digest `json:"digest,omitempty"`

	// URLs contains the source URLs of this content.
	URLs []string `json:"urls,omitempty"`

	// NOTE: Before adding a field here, please ensure that all
	// other options have been exhausted. Much of the type relationships
	// depend on the simplicity of this type.
}

// Manifest defines a schema2 manifest.
// github.com/docker/distribution/manifest/schema2/manifest.go
type Manifest struct {
	Versioned

	// Config references the image configuration as a blob.
	Config Descriptor `json:"config"`

	// Layers lists descriptors for the layers referenced by the
	// configuration.
	Layers []Descriptor `json:"layers"`
}
