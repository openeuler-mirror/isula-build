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
// Description: container transport related common functions tests

package container

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/storage/pkg/archive"
	"github.com/opencontainers/go-digest"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	mimetypes "isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
)

const (
	UnkonwArchive = 100
)

func TestCountDockerImageEmptyLayers(t *testing.T) {
	image := docker.Image{
		History: []docker.History{
			{
				Author:     "isula-1",
				CreatedBy:  "CMD",
				Comment:    "test-1",
				EmptyLayer: true,
			},
			{
				Author:     "isula-2",
				CreatedBy:  "RUN",
				Comment:    "test-2",
				EmptyLayer: false,
			},
			{
				Author:     "isula-3",
				CreatedBy:  "RUN",
				Comment:    "test-3",
				EmptyLayer: true,
			},
		},
	}

	cnt := countDockerImageEmptyLayers(image)
	assert.Equal(t, cnt, 1)
}

func TestGetImageLayerMIMEType(t *testing.T) {
	type testcase struct {
		name             string
		layerCompression archive.Compression
		expect           string
		isErr            bool
		errStr           string
	}

	testcases := []testcase{
		{
			name:             "gzip",
			layerCompression: archive.Gzip,
			expect:           mimetypes.DockerV2Schema2LayerMediaType,
		},
		{
			name:             "bzip2",
			layerCompression: archive.Bzip2,
			isErr:            true,
			errStr:           "is not support",
		},
		{
			name:             "xz",
			layerCompression: archive.Xz,
			isErr:            true,
			errStr:           "is not support",
		},
		{
			name:             "Zstd",
			layerCompression: archive.Zstd,
			isErr:            true,
			errStr:           "is not support",
		},
		{
			name:             "xxx",
			layerCompression: UnkonwArchive,
			isErr:            true,
			errStr:           "unknown compression format",
		},
	}

	for _, tc := range testcases {
		dmediaType, err := getImageLayerMIMEType(tc.layerCompression)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.ErrorContains(t, err, tc.errStr)
		} else {
			assert.Equal(t, dmediaType, tc.expect)
		}
	}
}

func TestEncodeConfigsAndManifests(t *testing.T) {
	type testcase struct {
		name           string
		dimage         docker.Image
		dmanifest      docker.Manifest
		manifestType   string
		expectConfig   string
		expectManifest string
	}

	testcases := []testcase{
		{
			name: "normal test",
			dimage: docker.Image{
				V1Image: docker.V1Image{
					ID:        "",
					Parent:    "",
					Comment:   "",
					Created:   time.Date(2017, 5, 12, 21, 36, 57, 81970000, time.UTC),
					Container: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
					ContainerConfig: docker.Config{
						Hostname:     "ab281de98ba0",
						User:         "",
						ExposedPorts: nil,
						Env:          []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
						Cmd:          []string{"sh"},
						Healthcheck: &docker.HealthConfig{
							Test:     []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
							Interval: time.Duration(interval),
							Timeout:  time.Duration(timeout),
						},
						Volumes:     nil,
						WorkingDir:  "",
						Entrypoint:  nil,
						OnBuild:     nil,
						Labels:      map[string]string{},
						StopSignal:  "",
						StopTimeout: nil,
						Shell:       nil,
					},
					DockerVersion: "1.12.1",
					Author:        "",
					Config: &docker.Config{
						Hostname: "ab281de98ba0",
						Env:      []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
						Cmd:      []string{"sh"},
						Healthcheck: &docker.HealthConfig{
							Test:     []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
							Interval: time.Duration(interval),
							Timeout:  time.Duration(timeout),
						},
						Labels: map[string]string{},
					},
					Architecture: "arm64",
					OS:           "linux",
					Size:         0,
				},
				Parent: "",
				RootFS: &docker.RootFS{Type: "layers", DiffIDs: []digest.Digest{}},
				History: []docker.History{
					{
						Created:   time.Date(2017, 5, 12, 21, 36, 57, 81970000, time.UTC),
						CreatedBy: "/bin/sh -c #(nop) ADD file:e9e6f86057e43a27b678a139b906091c3ecb1600b08ad17e80ff5ad56920c96e in / ",
					},
					{
						Created:    time.Date(2017, 5, 12, 21, 36, 57, 851043000, time.UTC),
						CreatedBy:  `/bin/sh -c #(nop)  CMD ["sh"]`,
						EmptyLayer: true,
					},
				},
			},

			dmanifest: docker.Manifest{
				Versioned: docker.Versioned{
					SchemaVersion: 2,
					MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
				},
				Config: docker.Descriptor{
					MediaType: "application/vnd.docker.container.image.v1+json",
					Size:      0,
					Digest:    "",
					URLs:      []string{},
				},
				Layers: []docker.Descriptor{
					{
						MediaType: "application/vnd.docker.image.rootfs.diff.tar",
						Size:      3522048,
						Digest:    "sha256:f91599b3986be816a2c74a3c05fda82e1b29f55eb12bc2b54fbfdfc3b5773edc",
						URLs:      []string{},
					},
					{
						MediaType: "application/vnd.docker.image.rootfs.diff.tar",
						Size:      3584,
						Digest:    "sha256:cfd168ba71d40c8608f2d8e1ece05367b178f528de892c46b895fba8d464055a",
						URLs:      []string{},
					},
				},
			},
			manifestType:   mimetypes.DockerV2Schema2MediaType,
			expectManifest: `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1156,"digest":"sha256:4fdcf74795b7ab3b55a8e86205e4a5497cac40948cf1f1e2e309184c75b7932d"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":3522048,"digest":"sha256:f91599b3986be816a2c74a3c05fda82e1b29f55eb12bc2b54fbfdfc3b5773edc"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar","size":3584,"digest":"sha256:cfd168ba71d40c8608f2d8e1ece05367b178f528de892c46b895fba8d464055a"}]}`,
		},
	}

	for _, tc := range testcases {
		_, manifest, err := encodeConfigsAndManifests(tc.dimage, tc.dmanifest, mimetypes.DockerV2Schema2MediaType)
		assert.NilError(t, err)
		assert.Equal(t, string(manifest), tc.expectManifest, tc.name)
	}
}

func checkDirTimeFunc(t *testing.T, dir string, ct time.Time) {
	entries, err := ioutil.ReadDir(dir)
	assert.NilError(t, err)
	for _, entry := range entries {
		srcPath := filepath.Join(dir, entry.Name())
		fi, err := os.Lstat(srcPath)
		assert.NilError(t, err)
		modTime := fi.ModTime()
		assert.DeepEqual(t, modTime, ct)
		if entry.IsDir() {
			checkDirTimeFunc(t, srcPath, ct)
		}
	}
}

func TestChMTimeDir(t *testing.T) {
	type testcase struct {
		name   string
		time   time.Time
		dir    *fs.Dir
		isErr  bool
		errStr string
	}

	testcases := []testcase{
		{
			name: "single file",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir:  fs.NewDir(t, t.Name(), fs.WithFile("file1", "isula")),
		},
		{
			name: "with multi file",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir:  fs.NewDir(t, t.Name(), fs.WithFile("file1", "isula"), fs.WithFile("file2", "isula")),
		},
		{
			name: "with subdir",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir: fs.NewDir(t, t.Name(), fs.WithDir("sub",
				fs.WithFiles(map[string]string{
					"file3": "contentb",
					"file4": "contentc",
				}),
				fs.WithMode(0705),
			)),
		},
		{
			name: "with symlink",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir: fs.NewDir(t, t.Name(), fs.WithFile("file1", "contenta", fs.WithMode(0400)),
				fs.WithFile("file2", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithFile("file5", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithSymlink("link1", "file1"),
				fs.WithDir("sub",
					fs.WithFiles(map[string]string{
						"file3": "contentb",
						"file4": "contentc",
					}),
					fs.WithMode(0705),
				),
			),
		},
		{
			name: "with symlink destination file not exit",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir: fs.NewDir(t, t.Name(), fs.WithFile("file1", "contenta", fs.WithMode(0400)),
				fs.WithFile("file2", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithFile("file5", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithSymlink("link1", "not_exist_file"),
			),
		},

		{
			name: "with symlink loop",
			time: time.Date(2006, 1, 7, 20, 34, 58, 651387237, time.UTC),
			dir: fs.NewDir(t, t.Name(), fs.WithFile("file1", "contenta", fs.WithMode(0400)),
				fs.WithFile("file2", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithFile("file5", "", fs.WithBytes([]byte{0, 1, 2})),
				fs.WithDir("sub",
					fs.WithFiles(map[string]string{
						"file3": "contentb",
						"file4": "contentc",
					}),
					fs.WithMode(0705),
					fs.WithSymlink("link1", "../sub"),
				),
			),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				tc.dir.Remove()
			}()

			err := ChMTimeDir(tc.dir.Path(), tc.time)
			assert.NilError(t, err)
			checkDirTimeFunc(t, tc.dir.Path(), tc.time)
		})
	}
}
