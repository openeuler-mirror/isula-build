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
	"bufio"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	constant "isula.org/isula-build"
)

const (
	// DefaultRSAKeySize is secure key length for RSA
	DefaultRSAKeySize = 2048
	// DefaultRSAKeyPath is the default directory to store rsa public key
	DefaultRSAKeyPath = "/etc/isula-build/isula-build.pub"
)

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
	if exist, err := IsExist(path); err != nil {
		return err
	} else if exist {
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
	if cErr := file.Close(); cErr != nil {
		return cErr
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
	key, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return rsa.PublicKey{}, errors.New("failed to find public key type")
	}

	return *key, nil
}

func checkSumReader(path string) (string, error) {
	const bufferSize = 32 * 1024 // 32KB

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", errors.Wrapf(err, "hash file failed")
	}
	defer func() {
		if cErr := file.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()
	buf := make([]byte, bufferSize)
	reader := bufio.NewReader(file)
	hasher := sha256.New()
	for {
		switch n, err := reader.Read(buf); err {
		case nil:
			hasher.Write(buf[:n])
		case io.EOF:
			return fmt.Sprintf("%x", hasher.Sum(nil)), nil
		default:
			return "", err
		}
	}
}

func hashFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if f, err := os.Stat(cleanPath); err != nil {
		return "", errors.Errorf("failed to stat file %q", cleanPath)
	} else if f.IsDir() {
		return "", errors.New("failed to hash directory")
	}

	return checkSumReader(path)
}

func hashDir(path string) (string, error) {
	var checkSum string
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		cleanPath := filepath.Clean(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if !info.IsDir() {
			fileHash, err := hashFile(cleanPath)
			if err != nil {
				return err
			}
			checkSum = fmt.Sprintf("%s%s", checkSum, fileHash)
		}
		return nil
	}); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(checkSum))), nil
}

// SHA256Sum will calculate sha256 checksum for path(file or directory)
// When calculate directory, each file of folder will be calculated and
// the checksum will be concatenated to next checksum until every file
// counted, the result will be used for final checksum calculation
func SHA256Sum(path string) (string, error) {
	if len(path) == 0 {
		return "", errors.New("failed to hash empty path")
	}
	path = filepath.Clean(path)
	f, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if f.IsDir() {
		return hashDir(path)
	}

	return hashFile(path)
}

// CheckSum will calculate the sha256sum for path and compare it with
// the target, if not match, return error
func CheckSum(path, target string) error {
	digest, err := SHA256Sum(path)
	if err != nil {
		return err
	}
	if digest != target {
		return errors.Errorf("check sum for path %s failed, got %s, want %s",
			path, digest, target)
	}
	return nil
}
