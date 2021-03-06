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
// Description: This file stores functions which used for aes encrypting and decrypting

package util

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"

	constant "isula.org/isula-build"
)

const (
	// CryptoKeyLen is secure key length for aes encryption and decryption(AES-256)
	CryptoKeyLen = 32
	// iteration is iteration count to hash
	iteration           = 409600
	aesKeyLenUpperBound = 32
	aesKeyLenLowerBound = 16
	// DefaultRSAKeySize is secure key length for RSA
	DefaultRSAKeySize = 2048
	// DefaultRSAKeyPath is the default directory to store rsa public key
	DefaultRSAKeyPath = "/etc/isula-build/isula-build.pub"
)

var (
	errGenCryptoKey = errors.New("generate crypto key failed")
)

// GenerateCryptoKey generates a random key with length s
// if used with AES, the input length can only be 16, 24, 32,
// which stands for AES-128, AES-192, or AES-256.
func GenerateCryptoKey(s int) ([]byte, error) {
	var size int
	if s >= aesKeyLenLowerBound && s <= aesKeyLenUpperBound {
		size = s
	} else {
		size = aesKeyLenLowerBound
	}
	key := make([]byte, size, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, errGenCryptoKey
	}

	return key, nil
}

// PBKDF2 is key derivation function to generate one way hash data
// if used with AES, the keyLen can only be 16, 24, 32
// which stands for AES-128, AES-192 or AES-256
// iteration is pre-set to 409600 and salt is generated by random key generator
func PBKDF2(password []byte, keyLen int, h func() hash.Hash) (string, error) {
	if len(password) == 0 {
		return "", errors.New("encrypt empty string failed")
	}
	salt, err := GenerateCryptoKey(CryptoKeyLen)
	if err != nil {
		return "", err
	}

	df := pbkdf2.Key(password, salt, iteration, keyLen, h)

	return hex.EncodeToString(df), nil
}

// EncryptAES encrypts plain text with AES encrypt algorithm(CFB)
func EncryptAES(data string, aeskey string) (string, error) {
	plainText := []byte(data)
	key, err := hex.DecodeString(aeskey)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv, err := GenerateCryptoKey(block.BlockSize())
	if err != nil {
		return "", errors.Errorf("generate rand data for iv failed: %v", err)
	}
	mode := cipher.NewCFBEncrypter(block, iv)
	encryptData := make([]byte, len(plainText), len(plainText))
	mode.XORKeyStream(encryptData, plainText)
	encryptData = append(iv, encryptData...)

	return hex.EncodeToString(encryptData), nil
}

// DecryptAES decrypts text with AES decrypt algorithm(CFB)
func DecryptAES(data string, aeskey string) (string, error) {
	key, err := hex.DecodeString(aeskey)
	if err != nil {
		return "", err
	}

	cipherText, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(cipherText) <= block.BlockSize() {
		return "", errors.Errorf("invalid cipher text length %v, it must larger than %v", len(cipherText), block.BlockSize())
	}

	decrypter := cipher.NewCFBDecrypter(block, cipherText[:block.BlockSize()])
	decryptData := make([]byte, len(cipherText)-block.BlockSize(), len(cipherText)-block.BlockSize())
	decrypter.XORKeyStream(decryptData, cipherText[block.BlockSize():])

	return string(decryptData), nil
}

// GenerateRSAKey generates a RAS key pair with key size s
// the recommend key size is 4096 and which will be use when
// key size is less than it
func GenerateRSAKey(keySize int) (*rsa.PrivateKey, error) {
	if keySize <= DefaultRSAKeySize {
		keySize = DefaultRSAKeySize
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, errors.Errorf("generate rsa key pair failed: %v", err)
	}

	return privateKey, nil
}

// EncryptRSA encrypts text with RSA public key
// the hash function(ordinary one) need to be same level with decrypt end
func EncryptRSA(data string, key rsa.PublicKey, h hash.Hash) (string, error) {
	cipherText, err := rsa.EncryptOAEP(h, rand.Reader, &key, []byte(data), nil)
	if err != nil {
		return "", errors.Errorf("encryption failed: %v", err)
	}

	return hex.EncodeToString(cipherText), nil
}

// DecryptRSA decrypts cipher text with RSA private key
// the hash function(crypto one) need to be same level with encrypt end
func DecryptRSA(data string, key *rsa.PrivateKey, h crypto.Hash) (string, error) {
	msg, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}
	plainText, errDec := key.Decrypt(nil, msg, &rsa.OAEPOptions{Hash: h, Label: nil})
	if errDec != nil {
		return "", errors.Errorf("decryption failed: %v", err)
	}

	return string(plainText), nil
}

// GenRSAPublicKeyFile store public key from rsa key pair into local file
func GenRSAPublicKeyFile(key *rsa.PrivateKey, path string) error {
	if IsExist(path) {
		if err := os.Remove(path); err != nil {
			return errors.Errorf("failed to delete the residual key file: %v", err)
		}
	}
	publicKey := &key.PublicKey
	stream, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: stream,
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := os.Chmod(path, constant.DefaultReadOnlyFileMode); err != nil {
		return err
	}
	if err := pem.Encode(file, block); err != nil {
		return err
	}

	return nil
}

// ReadPublicKey gets public key from key file
func ReadPublicKey(path string) (rsa.PublicKey, error) {
	keyFile, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return rsa.PublicKey{}, err
	}
	block, _ := pem.Decode(keyFile)
	if block == nil {
		return rsa.PublicKey{}, errors.New("decoding public key failed")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return rsa.PublicKey{}, err
	}
	key := pubInterface.(*rsa.PublicKey)

	return *key, nil
}
