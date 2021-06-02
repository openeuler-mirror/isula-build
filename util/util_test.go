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
// Create: 2020-04-01
// Description: common functions tests

package util

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
)

func TestGetIgnorePatternMatcher(t *testing.T) {
	contextDir := "/tmp/isula-build/contextDir"
	ignores := []string{"test*", "a", "b"}
	matcher, err := GetIgnorePatternMatcher(ignores, contextDir, "")
	assert.NilError(t, err)
	assert.Equal(t, matcher != nil, true)
	result, err := matcher.MatchesResult(contextDir + "/test1")
	assert.NilError(t, err)
	assert.Equal(t, result.IsMatched(), true)

	result, err = matcher.MatchesResult(contextDir + "/tes")
	assert.NilError(t, err)
	assert.Equal(t, result.IsMatched(), false)
}

func TestCopyURLResource(t *testing.T) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "It's my return!")
	})
	go func() {
		http.ListenAndServe(":12345", nil)
	}()
	time.Sleep(time.Second)

	url := "http://localhost:12345/"
	dest := "/tmp/file-for-test"
	uid := 1000
	gid := 1000
	err := CopyURLResource(context.Background(), url, dest, uid, gid)
	assert.NilError(t, err)

	f, err := os.Stat(dest)
	assert.NilError(t, err)

	stat, ok := f.Sys().(*syscall.Stat_t)
	assert.Equal(t, ok, true)
	assert.Equal(t, int(stat.Uid), uid)
	assert.Equal(t, int(stat.Gid), gid)

	err = os.Remove(dest)
	assert.NilError(t, err)
}

func TestCopyFile(t *testing.T) {
	src := fmt.Sprintf("/tmp/test-%d", rand.Int())
	f, err := os.Create(src)
	defer func() {
		f.Close()
		err = os.Remove(src)
		assert.NilError(t, err)
	}()

	var testcases = struct {
		attrName []string
		attr     []string
	}{
		attrName: []string{"security.smack1", "security.ima2", "security.evm3"},
		attr:     []string{"smack1", "ima2", "evm3"},
	}

	for index := range testcases.attrName {
		err := system.Lsetxattr(src, testcases.attrName[index], []byte(testcases.attr[index]), 0)
		assert.NilError(t, err)
	}

	assert.NilError(t, err)
	_, err = f.Write([]byte("This is a test file."))
	assert.NilError(t, err)

	dir := fmt.Sprintf("/tmp/test2-%d/", rand.Int())
	dest := dir + "test"
	err = CopyFile(src, dest, idtools.IDPair{})
	defer func() {
		err = os.RemoveAll(dir)
		assert.NilError(t, err)
	}()
	assert.NilError(t, err)

	srcFileInfo, err := os.Stat(src)
	assert.NilError(t, err)
	destFileInfo, err := os.Stat(dest)
	assert.NilError(t, err)
	assert.Equal(t, srcFileInfo.Size(), destFileInfo.Size())
	assert.Equal(t, srcFileInfo.Mode(), destFileInfo.Mode())
	for index := range testcases.attrName {
		attrValue, err := system.Lgetxattr(dest, testcases.attrName[index])
		assert.NilError(t, err)
		assert.Equal(t, bytes.Compare(attrValue[:len(testcases.attr[index])],
			[]byte(testcases.attr[index])), 0)
	}
}

func TestGenerateCryptoNum(t *testing.T) {
	var testcases = []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "TC1 - normal case",
			length:  12,
			wantErr: false,
		},
		{
			name:    "TC2 - wrong input length",
			length:  20,
			wantErr: true,
		},
		{
			name:    "TC3 - wrong input length",
			length:  -1,
			wantErr: true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCryptoNum(tt.length)
			if err == nil {
				assert.Equal(t, len(result), tt.length)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestCopyXattrs(t *testing.T) {
	var testcases = []struct {
		name     string
		attrName []string
		attr     []string
		wantAttr []string
	}{
		{
			name:     "1",
			attrName: []string{"security.smack1", "security.ima2", "security.evm3"},
			attr:     []string{"c", "d", "e"},
			wantAttr: []string{"c", "d", "e"},
		},
		{
			name:     "2",
			attrName: []string{"security.selinux55", "trusted.ppp"},
			attr:     []string{"system_ddu:object_wdr:tmp_t:saa83211", "www"},
			wantAttr: []string{"system_ddu:object_wdr:tmp_t:saa83211", ""},
		},
		{
			name:     "3",
			attrName: []string{"security.ima1", "trusted.p1", "trusted.p2"},
			attr:     []string{"wwww", "t1", "t2"},
			wantAttr: []string{"wwww", "", ""},
		},
		{
			name:     "4",
			attrName: []string{"security.ima1", "trusted.xz1"},
			attr:     []string{"ima", "trust"},
			wantAttr: []string{"ima", ""},
		},
		{
			name:     "5",
			attrName: []string{"security.smack1", "security.evm2"},
			attr:     []string{"12313122321332121333333312", "123131223333333333333312"},
			wantAttr: []string{"12313122321332121333333312", "123131223333333333333312"},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			src := "/tmp/attrSrcFile"
			dest := "/tmp/attrDestFile"
			f, err := os.Create(src)
			f2, err2 := os.Create(dest)

			defer func() {
				f.Close()
				f2.Close()
				os.Remove(src)
				os.Remove(dest)
			}()
			assert.NilError(t, err)
			assert.NilError(t, err2)

			for index := range tt.attrName {
				err := system.Lsetxattr(src, tt.attrName[index], []byte(tt.attr[index]), 0)
				assert.NilError(t, err)
			}

			err = CopyXattrs(src, dest)
			assert.NilError(t, err)

			for index := range tt.attrName {
				attrValue, err := system.Lgetxattr(dest, tt.attrName[index])
				if tt.wantAttr[index] == "" {
					assert.Equal(t, len(attrValue), 0)
					continue
				}
				assert.NilError(t, err)
				assert.Equal(t, bytes.Compare(attrValue[:len(tt.attr[index])], []byte(tt.wantAttr[index])), 0)
			}

		})
	}

}

func TestValidateSignal(t *testing.T) {
	type args struct {
		sigStr string
	}
	tests := []struct {
		name    string
		args    args
		want    syscall.Signal
		wantErr bool
	}{
		{
			name:    "TC1 - normal case with integer",
			args:    args{sigStr: "9"},
			want:    syscall.Signal(9),
			wantErr: false,
		},
		{
			name:    "TC2 - normal case with signal name",
			args:    args{sigStr: "SIGKILL"},
			want:    syscall.Signal(9),
			wantErr: false,
		},
		{
			name:    "TC3 - abnormal case with invalid signal name",
			args:    args{sigStr: "aaa"},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "TC4 - abnormal case with invalid signal value",
			args:    args{sigStr: "65"},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "TC5 - abnormal case with invalid signal value",
			args:    args{sigStr: "0"},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateSignal(tt.args.sigStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateSignal() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestSetDaemonLock(t *testing.T) {
	root := "/tmp/this_is_a_test_folder"
	name := "test.lock"
	lockPath := filepath.Join(root, name)

	// when folder is not exist, daemon lock is not supposed to be set
	_, err := SetDaemonLock(root, name)
	assert.ErrorContains(t, err, "no such file or directory")

	// create lockfile
	err = os.Mkdir(root, constant.DefaultRootDirMode)
	defer os.RemoveAll(root)
	assert.NilError(t, err)
	f, err := os.Create(lockPath)
	assert.NilError(t, err)
	defer f.Close()
	// set daemon lock successful
	_, err = SetDaemonLock(root, name)
	assert.NilError(t, err)

	// set daemon lock twice will fail
	_, err = SetDaemonLock(root, name)
	assert.ErrorContains(t, err, "check if there is another daemon running")
}

func TestGenerateNonCryptoID(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			name: "TC1 - generate id",
			want:64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateNonCryptoID(); len(got) != tt.want {
				t.Errorf("GenerateNonCryptoID() = %v, want %v", got, tt.want)
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
			format:  constant.DockerTransport,
			wantErr: false,
		},
		{
			name:    "oci image format",
			format:  constant.OCITransport,
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

func TestIsClientExporter(t *testing.T) {
	testcases := []struct {
		name       string
		exporter   string
		wantResult bool
	}{
		{
			name:       "normal docker archive exporter",
			exporter:   constant.DockerArchiveTransport,
			wantResult: true,
		},
		{
			name:       "normal oci archive exporter",
			exporter:   constant.OCIArchiveTransport,
			wantResult: true,
		},
		{
			name:       "normal isulad exporter",
			exporter:   constant.IsuladTransport,
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
