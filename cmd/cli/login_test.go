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
// Create: 2020-07-20
// Description: This file is used for testing command login

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestNewLoginCmd(t *testing.T) {
	loginCmd := NewLoginCmd()
	loginCmd.SetArgs(strings.Split("test.org --username testuser --password-stdin", " "))
	err := loginCmd.Execute()
	args := []string{"test.org"}
	err = loginCommand(loginCmd, args)
	if err != nil {
		assert.ErrorContains(t, err, "isula_build.sock")
	}
}

func TestGetPassFromInput(t *testing.T) {
	type args struct {
		f passReader
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TC1 - normal input",
			args: args{f: func() ([]byte, error) {
				return []byte("aaa"), nil
			}},
			wantErr: false,
		},
		{
			name: "TC2 - abnormal input with error",
			args: args{f: func() ([]byte, error) {
				return nil, errors.New("error read password")
			}},
			wantErr: true,
		},
		{
			name: "TC3 - abnormal input with length more than 128",
			args: args{func() ([]byte, error) {
				return bytes.Repeat([]byte("a"), 129), nil
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getPassFromInput(tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("getPassFromInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPassFromStdin(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TC1 - normal input",
			args:    args{r: strings.NewReader("aaa")},
			wantErr: false,
		},
		{
			name:    "TC2 - empty input",
			args:    args{r: strings.NewReader("")},
			wantErr: true,
		},
		{
			name:    "TC2 - abnormal input length",
			args:    args{r: strings.NewReader(string(bytes.Repeat([]byte("a"), 129)))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getPassFromStdin(tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("getPassFromStdin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptOpts(t *testing.T) {
	err := encryptOpts()
	assert.NilError(t, err)
}

func TestRunLogin(t *testing.T) {
	type testcase struct {
		name      string
		server    string
		username  string
		password  string
		wantErr   bool
		errString string
	}
	var testcases = []testcase{
		{
			name:     "TC1 - normal case",
			server:   "test.org",
			username: "testUser",
			password: "testPass",
			wantErr:  false,
		},
		{
			name:      "TC2 - abnormal case with empty server",
			server:    "",
			wantErr:   true,
			errString: "empty server address",
		},
		{
			name:     "TC3 - abnormal case with empty password",
			server:   "test.org",
			username: "testUser",
			password: "",
			wantErr:  true,
		},
	}
	for _, tc := range testcases {
		ctx := context.Background()
		mockD := newMockDaemon()
		cli := newMockClient(&mockGrpcClient{loginFunc: mockD.login})

		c := NewLoginCmd()
		loginOpts = loginOptions{
			server:   tc.server,
			username: tc.username,
			password: tc.password,
		}

		_, err := runLogin(ctx, &cli, c)
		assert.Equal(t, err != nil, tc.wantErr, "Failed at [%s], err: %v", tc.name, err)
		if err != nil {
			assert.ErrorContains(t, err, tc.errString)
		}
	}
}
