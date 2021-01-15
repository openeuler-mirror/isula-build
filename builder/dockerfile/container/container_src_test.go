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
// Description: container image source related functions tests

package container

import (
	"context"
	"os"
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestClose(t *testing.T) {
	cis := containerImageSource{
		path: fs.NewDir(t, "blob").Path(),
	}
	cis.Close()
	_, err := os.Stat(cis.path)
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestReference(t *testing.T) {
	var name reference.Named
	metadata := &ReferenceMetadata{
		Name:        name,
		CreatedBy:   "isula",
		Dconfig:     []byte("isula-builder"),
		ContainerID: "e6587b2dbfd56b5ce2e64dd7933ba04886bff86836dec5f09ce59d599df012fe",
		LayerID:     "dacfba0cd5c0d28f33d41fb9a9c8bf2b0c53689da136aeba6dfecf347125fa23",
	}

	imageRef := NewContainerReference(&localStore, metadata, false)
	cis := containerImageSource{
		ref: &imageRef,
	}
	r := cis.Reference()
	transport := r.StringWithinTransport()
	assert.Equal(t, transport, "container")
}

func TestGetSignatures(t *testing.T) {
	type testcase struct {
		name         string
		digest       *digest.Digest
		manifest     []byte
		manifestType string
		isErr        bool
		errStr       string
	}
	d := digest.SHA256.FromString("isula")
	var testcases = []testcase{
		{
			name:   "with digest",
			digest: &d,
			isErr:  true,
		},
		{
			name: "with nil digest",
		},
	}

	for _, tc := range testcases {
		cis := containerImageSource{}
		signature, err := cis.GetSignatures(context.TODO(), tc.digest)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.ErrorContains(t, err, "not support to list the signatures")
		}
		if err == nil {
			assert.DeepEqual(t, signature, [][]uint8(nil))
		}
	}
}

func TestGetManifest(t *testing.T) {
	type testcase struct {
		name         string
		digest       *digest.Digest
		manifest     []byte
		manifestType string
		isErr        bool
		errStr       string
	}
	d := digest.SHA256.FromString("isula")
	var testcases = []testcase{
		{
			name:   "with digest",
			digest: &d,
			isErr:  true,
		}, {
			name: "with nil digest",
		},
	}

	for _, tc := range testcases {
		cis := containerImageSource{
			manifest:     []byte("6d47a9873783f7bf23773f0cf60c67cef295d451f56b8b79fe3a1ea217a4bf98"),
			manifestType: manifest.DockerV2Schema2MediaType,
		}
		manifest, manifestType, err := cis.GetManifest(context.TODO(), tc.digest)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.ErrorContains(t, err, "not support list the manifest")
		}
		if err == nil {
			assert.Equal(t, string(manifest), string(cis.manifest))
			assert.Equal(t, manifestType, cis.manifestType)
		}
	}
}

func TestLayerInfosForCopy(t *testing.T) {
	cis := containerImageSource{
		manifest:     []byte("6d47a9873783f7bf23773f0cf60c67cef295d451f56b8b79fe3a1ea217a4bf98"),
		manifestType: manifest.DockerV2Schema2MediaType,
	}
	info, err := cis.LayerInfosForCopy(context.TODO(), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, info, []types.BlobInfo(nil))
}

func TestHasThreadSafeGetBlob(t *testing.T) {
	cis := containerImageSource{}
	b := cis.HasThreadSafeGetBlob()
	assert.Equal(t, b, false)
}

func TestGetBlob(t *testing.T) {
	type testcase struct {
		name        string
		digestStr   string
		hasBlobFile bool
		isErr       bool
		errStr      string
		expectSize  int64
	}
	var testcases = []testcase{
		{
			name:       "digest equal",
			digestStr:  "digest equal",
			expectSize: 12,
		},
		{
			name:      "digest is not equal and blob file not exist",
			digestStr: "digest",
			isErr:     true,
			errStr:    "no such file or directory",
		},
		{
			name:        "has blob file",
			digestStr:   "digest",
			hasBlobFile: true,
			expectSize:  12,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			d := digest.SHA256.FromString(tc.name)
			cis := containerImageSource{
				configDigest: d,
				config:       []byte(tc.name),
			}
			blob := types.BlobInfo{
				Digest: digest.SHA256.FromString(tc.digestStr),
			}

			if tc.hasBlobFile {
				dirCtx := fs.NewDir(t, t.Name(), fs.WithFile(blob.Digest.String(), "blob-content"))
				cis.path = dirCtx.Path()
				defer dirCtx.Remove()
			}

			_, size, err := cis.GetBlob(context.TODO(), blob, nil)
			assert.Equal(t, err != nil, tc.isErr, tc.name)
			if err != nil {
				assert.ErrorContains(t, err, tc.errStr)
			}
			if err == nil {
				assert.Equal(t, tc.expectSize, size, tc.name)
			}
		})
	}
}
