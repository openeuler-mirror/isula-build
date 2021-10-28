// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-06-02
// Description: aes encrypt and decrypt testing

package util

import (
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	constant "isula.org/isula-build"
)

const (
	maxRepeatTime = 1000000
)

func TestAES(t *testing.T) {
	var testcases = []struct {
		name    string
		length  int
		wantErr bool
		text    string
		hash    func() hash.Hash
	}{
		{
			name:    "TC1 - normal case with key length 16",
			length:  16,
			text:    "abcdefghijklmnopqrstuvwxyz",
			hash:    sha256.New,
			wantErr: false,
		},
		{
			name:    "TC2 - normal case with key length 24",
			length:  24,
			text:    "1234567890",
			hash:    sha256.New,
			wantErr: false,
		},
		{
			name:    "TC3 - normal case with key length 32",
			length:  32,
			text:    "!@#$%^&*()_+:><?",
			hash:    sha256.New,
			wantErr: false,
		},
		{
			name:    "TC4 - normal case with sha1",
			length:  32,
			text:    "1234567890",
			hash:    sha1.New,
			wantErr: false,
		},
		{
			name:    "TC5 - normal case with sha256",
			length:  32,
			text:    "abcdefghijklmnopqrstuvwxyz",
			hash:    sha512.New,
			wantErr: false,
		},
		{
			name:    "TC6 - abnormal case with invalid key length 0",
			length:  0,
			text:    "!@#$%^&*()_+:><?",
			hash:    sha256.New,
			wantErr: true,
		},
		{
			name:    "TC7 - abnormal case with invalid ken length 63",
			length:  63,
			text:    "This is test 7",
			hash:    sha256.New,
			wantErr: true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			oriKey, err := GenerateCryptoKey(tt.length)
			key, err := PBKDF2(oriKey, tt.length, tt.hash)
			encryptData, err := EncryptAES(tt.text, key)
			decryptData, err := DecryptAES(encryptData, key)
			if err == nil {
				assert.Equal(t, tt.text, decryptData)
				assert.Assert(t, string(oriKey) != key)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestRSA(t *testing.T) {
	type args struct {
		data string
		he   hash.Hash
		hd   crypto.Hash
	}
	tests := []struct {
		name      string
		args      args
		wantEnErr bool
		wantDeErr bool
	}{
		{
			name: "TC1 - normal case with sha512",
			args: args{
				data: "This is a plain text",
				he:   sha512.New(),
				hd:   crypto.SHA512,
			},
		},
		{
			name: "TC2 - normal case with sha256",
			args: args{
				data: "This is a plain text",
				he:   sha256.New(),
				hd:   crypto.SHA256,
			},
		},
		{
			name: "TC3 - normal case with sha1",
			args: args{
				data: "This is a plain text",
				he:   sha1.New(),
				hd:   crypto.SHA1,
			},
		},
		{
			name: "TC4 - normal case with empty data to encrypt",
			args: args{
				data: "",
				he:   sha512.New(),
				hd:   crypto.SHA512,
			},
		},
		{
			name: "TC5 - abnormal case with different hash function between encryption and decryption",
			args: args{
				data: "This is plain text",
				he:   sha512.New(),
				hd:   crypto.SHA256,
			},
			wantDeErr: true,
		},
		{
			name: "TC6 - abnormal case with too long plain text",
			args: args{
				data: strings.Repeat("a", maxRepeatTime),
				he:   sha512.New(),
				hd:   crypto.SHA512,
			},
			wantEnErr: true,
			wantDeErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateRSAKey(DefaultRSAKeySize)
			assert.NilError(t, err)
			cipherText, err := EncryptRSA(tt.args.data, key.PublicKey, tt.args.he)
			if (err != nil) != tt.wantEnErr {
				t.Errorf("EncryptRSA() error = %v, wantErr %v", err, tt.wantEnErr)
				return
			}
			plainText, err := DecryptRSA(cipherText, key, tt.args.hd)
			if (err != nil) != tt.wantDeErr {
				t.Errorf("DecryptRSA() error = %v, wantErr %v", err, tt.wantDeErr)
				return
			}
			if err == nil {
				if plainText != tt.args.data {
					t.Errorf("EncryptRSA() got = %v, want %v", plainText, tt.args.data)
				}
			}
		})
	}
}

func TestGenRSAPubKey(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	keyPath := filepath.Join(tmpDir.Path(), "isula-build.pub")
	rsaKey, err := GenerateRSAKey(DefaultRSAKeySize)
	assert.NilError(t, err)
	err = GenRSAPublicKeyFile(rsaKey, keyPath)
	assert.NilError(t, err)
	// when there already has key
	err = GenRSAPublicKeyFile(rsaKey, keyPath)
	assert.NilError(t, err)
}

func benchmarkGenerateRSAKey(scale int, b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		GenerateRSAKey(scale)
	}
}

func BenchmarkGenerateRSAKey2048(b *testing.B) { benchmarkGenerateRSAKey(2048, b) }
func BenchmarkGenerateRSAKey3072(b *testing.B) { benchmarkGenerateRSAKey(3072, b) }
func BenchmarkGenerateRSAKey4096(b *testing.B) { benchmarkGenerateRSAKey(4096, b) }

func TestHashFile(t *testing.T) {
	emptyFile := fs.NewFile(t, t.Name())
	defer emptyFile.Remove()
	fileWithContent := fs.NewFile(t, t.Name())
	err := ioutil.WriteFile(fileWithContent.Path(), []byte("hello"), constant.DefaultRootFileMode)
	assert.NilError(t, err)
	defer fileWithContent.Remove()
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TC-hash empty file",
			args: args{path: emptyFile.Path()},
			// empty file sha256sum always is
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "TC-hash file with content",
			args: args{path: fileWithContent.Path()},
			want: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:    "TC-hash file with empty path",
			wantErr: true,
		},
		{
			name:    "TC-hash file with invalid path",
			args:    args{path: "path not exist"},
			wantErr: true,
		},
		{
			name:    "TC-hash file with directory path",
			args:    args{path: dir.Path()},
			wantErr: true,
		},
		{
			name:    "TC-hash file with special device",
			args:    args{path: "/dev/cdrom"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hashFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("hashFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hashFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashDir(t *testing.T) {
	root := fs.NewDir(t, t.Name())
	defer root.Remove()

	rootSub1 := root.Join("sub1")
	os.MkdirAll(rootSub1, constant.DefaultRootDirMode)
	defer os.RemoveAll(rootSub1)
	rootSub1File := filepath.Join(rootSub1, "rootSub1File")
	ioutil.WriteFile(rootSub1File, []byte("hello1"), constant.DefaultRootFileMode)
	defer os.RemoveAll(rootSub1File)

	rootSub11 := filepath.Join(rootSub1, "sub11")
	os.MkdirAll(rootSub11, constant.DefaultRootDirMode)
	defer os.RemoveAll(rootSub11)
	rootSub11File := filepath.Join(rootSub11, "rootSub11File")
	ioutil.WriteFile(rootSub11File, []byte("hello11"), constant.DefaultRootFileMode)
	defer os.RemoveAll(rootSub11File)

	emptyDir := fs.NewDir(t, t.Name())
	defer emptyDir.Remove()
	emptyFile := root.Join("empty.tar")
	_, err := os.Create(emptyFile)
	assert.NilError(t, err)
	defer os.RemoveAll(emptyFile)

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TC-hash empty dir",
			args: args{path: emptyDir.Path()},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "TC-hash not exist dir",
			args:    args{path: "path not exist"},
			wantErr: true,
		},
		{
			name: "TC-hash multiple dirs",
			args: args{path: root.Path()},
			want: "bdaaa88766b974876a14d85620b5a26795735c332445783a3a067e0052a59478",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hashDir(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("hashDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hashDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSHA256Sum(t *testing.T) {
	root := fs.NewDir(t, t.Name())
	defer root.Remove()

	rootSub1 := root.Join("sub1")
	os.MkdirAll(rootSub1, constant.DefaultRootDirMode)
	defer os.RemoveAll(rootSub1)
	rootSub1File := filepath.Join(rootSub1, "rootSub1File")
	ioutil.WriteFile(rootSub1File, []byte("hello1"), constant.DefaultRootFileMode)
	defer os.RemoveAll(rootSub1File)

	emptyDir := fs.NewDir(t, t.Name())
	defer emptyDir.Remove()
	emptyFile := root.Join("empty.tar")
	_, err := os.Create(emptyFile)
	assert.NilError(t, err)
	defer os.RemoveAll(emptyFile)

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TC-for dir",
			args: args{path: root.Path()},
			want: "6a29015d578de92eabad6b20b3e3c0d4df521b03728cb4ee5667b15742154646",
		},
		{
			name: "TC-for file only",
			args: args{path: emptyFile},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "TC-for invalid file",
			args:    args{path: "/dev/cdrom"},
			wantErr: true,
		},
		{
			name:    "TC-for path not exist",
			args:    args{path: "path not exist"},
			wantErr: true,
		},
		{
			name:    "TC-for empty path",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SHA256Sum(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SHA256Sum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SHA256Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSum(t *testing.T) {
	emptyFile := fs.NewFile(t, t.Name())
	defer emptyFile.Remove()

	type args struct {
		path   string
		target string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TC-normal case",
			args: args{
				path:   emptyFile.Path(),
				target: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
		{
			name:    "TC-check sum failed",
			args:    args{path: emptyFile.Path(), target: "wrong"},
			wantErr: true,
		},
		{
			name:    "TC-empty path",
			args:    args{target: "wrong"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckSum(tt.args.path, tt.args.target); (err != nil) != tt.wantErr {
				t.Errorf("CheckSum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
