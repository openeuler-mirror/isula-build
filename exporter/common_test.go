// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2021-01-04
// Description: package exporter test functions

package exporter

import (
	"strings"
	"testing"

	"github.com/containers/image/v5/manifest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestFormatTransport(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	ociArchiveFilePath := tmpDir.Join("test.tar")
	defer tmpDir.Remove()

	testcases := []struct {
		name      string
		transport string
		path      string
		result    string
	}{
		{
			name:      "docker format transport",
			transport: DockerTransport,
			path:      "registry.example.com/library/image:test",
			result:    "docker://registry.example.com/library/image:test",
		},
		{
			name:      "oci-archive format transport",
			transport: OCIArchiveTransport,
			path:      ociArchiveFilePath,
			result:    "oci-archive:",
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			if testcase.name == "docker" {
				assert.Equal(t, FormatTransport(testcase.transport, testcase.path), testcase.result)
			}
			if testcase.name == "oci-archive" {
				assert.Assert(t, true, strings.Contains(FormatTransport(testcase.transport, testcase.path), testcase.result))
			}
		})
	}
}

func TestCheckImageFormat(t *testing.T) {
	testcases := []struct {
		name      string
		format    string
		wantErr   bool
		errString string
	}{
		{
			name:    "docker image format",
			format:  DockerTransport,
			wantErr: false,
		},
		{
			name:    "oci image format",
			format:  OCITransport,
			wantErr: false,
		},
		{
			name:    "unknown image format",
			format:  "you guess",
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckImageFormat(tc.format)
			if tc.wantErr {
				assert.Error(t, err, "wrong image format provided")
				return
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
}

func TestCheckArchiveFormat(t *testing.T) {
	testcases := []struct {
		name      string
		format    string
		wantErr   bool
		errString string
	}{
		{
			name:    "docker-archive image format",
			format:  DockerArchiveTransport,
			wantErr: false,
		},
		{
			name:    "oci-archive imagee format",
			format:  OCIArchiveTransport,
			wantErr: false,
		},
		{
			name:    "unknown image format",
			format:  "you guess",
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckArchiveFormat(tc.format)
			if tc.wantErr {
				assert.Error(t, err, "wrong image format provided")
				return
			}
			if !tc.wantErr {
				assert.NilError(t, err)
			}
		})
	}
}

func TestGetManifestType(t *testing.T) {
	testcases := []struct {
		name      string
		format    string
		manifest  string
		wantErr   bool
		errString string
	}{
		{
			name:     "docker format manifest type",
			format:   DockerTransport,
			manifest: manifest.DockerV2Schema2MediaType,
			wantErr:  false,
		},
		{
			name:     "oci format manifest type",
			format:   OCITransport,
			manifest: imgspecv1.MediaTypeImageManifest,
			wantErr:  false,
		},
		{
			name:    "unknown format manifest type",
			format:  "unkown",
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := GetManifestType(tc.format)
			if tc.wantErr {
				assert.ErrorContains(t, err, "unknown format")
				return
			}
			if !tc.wantErr {
				assert.Equal(t, manifest, tc.manifest)
			}
		})
	}
}

func TestIsClientExporter(t *testing.T) {
	testcases := []struct {
		name       string
		exporter   string
		wantResult bool
	}{
		{
			name:       "normal docker archive exporter",
			exporter:   DockerArchiveTransport,
			wantResult: true,
		},
		{
			name:       "normal oci archive exporter",
			exporter:   OCIArchiveTransport,
			wantResult: true,
		},
		{
			name:       "normal isulad exporter",
			exporter:   IsuladTransport,
			wantResult: true,
		},
		{
			name:       "abnormal unkown",
			exporter:   "unkown",
			wantResult: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			isExporter := IsClientExporter(tc.exporter)
			if isExporter != tc.wantResult {
				t.Fatal("test client exporter failed")
			}
		})
	}
}
