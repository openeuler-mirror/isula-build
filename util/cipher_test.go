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
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
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
