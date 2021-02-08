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
// Description: container image source related functions

package container

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	"isula.org/isula-build/store"
)

type containerImageSource struct {
	ref          *Reference
	path         string
	containerID  string
	layerID      string
	manifestType string
	config       []byte
	manifest     []byte
	store        *store.Store
	compression  archive.Compression
	configDigest digest.Digest
	exporting    bool
}

// Close removes the blob directory associated with the containerImageSource
func (i *containerImageSource) Close() error {
	err := os.RemoveAll(i.path)
	if err != nil {
		return errors.Wrapf(err, "remove the layer's blob directory %q failed", i.path)
	}
	return nil
}

// Reference returns the reference used to set up this source
func (i *containerImageSource) Reference() types.ImageReference {
	return i.ref
}

// GetSignatures used to get the image's signatures, but containerImageSource not
// support to list it
func (i *containerImageSource) GetSignatures(ctx context.Context, instanceDigest *digest.Digest) ([][]byte, error) {
	if instanceDigest != nil {
		return nil, errors.Errorf("containerImageSource does not support to list the signatures")
	}
	return nil, nil
}

// GetManifest returns the image's manifest along with its MIME type
func (i *containerImageSource) GetManifest(ctx context.Context, instanceDigest *digest.Digest) ([]byte, string, error) {
	if instanceDigest != nil {
		return nil, "", errors.Errorf("containerImageSource does not support list the manifest")
	}
	return i.manifest, i.manifestType, nil
}

// LayerInfosForCopy always return nil here meaning the values in the manifest are fine
func (i *containerImageSource) LayerInfosForCopy(ctx context.Context, instanceDigest *digest.Digest) ([]types.BlobInfo, error) {
	return nil, nil
}

// HasThreadSafeGetBlob always return nil here indicates the GetBlob can not be executed concurrently
func (i *containerImageSource) HasThreadSafeGetBlob() bool {
	return false
}

// GetBlob returns a stream for the specified blob, and the blob’s size
func (i *containerImageSource) GetBlob(ctx context.Context, blob types.BlobInfo, _ types.BlobInfoCache) (io.ReadCloser, int64, error) {
	if blob.Digest == i.configDigest {
		reader := bytes.NewReader(i.config)
		return ioutil.NopCloser(reader), reader.Size(), nil
	}

	blobFile := filepath.Join(i.path, blob.Digest.String())
	st, err := os.Stat(blobFile)
	if err != nil && os.IsNotExist(err) {
		return nil, -1, errors.Wrapf(err, "blob file %q is not exit", blobFile)
	}

	layerFile, err := os.OpenFile(filepath.Clean(blobFile), os.O_RDONLY, constant.DefaultRootFileMode)
	if err != nil {
		return nil, -1, errors.Wrapf(err, "open the blob file %q failed", blobFile)
	}

	return layerFile, st.Size(), nil
}
