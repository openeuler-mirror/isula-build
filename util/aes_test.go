// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// iSula-Kits licensed under the Mulan PSL v2.
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
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"testing"

	"gotest.tools/assert"
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
