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
// Description: This file is used for testing command logout

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestRunLogout(t *testing.T) {
	type testcase struct {
		name      string
		all       bool
		server    string
		errString string
		wantErr   bool
	}
	var testcases = []testcase{
		{
			name:    "TC1 - normal case",
			all:     false,
			server:  "test.org",
			wantErr: false,
		},
		{
			name:    "TC2 - normal case with -a flag",
			all:     true,
			server:  "test.org",
			wantErr: false,
		},
		{
			name:    "TC3 - abnormal case with empty server name",
			all:     false,
			server:  "",
			wantErr: true,
		},
		{
			name:    "TC4 - abnormal case with empty server name and -a flag",
			all:     true,
			server:  "",
			wantErr: true,
		},
		{
			name:    "TC5 - abnormal case with server name larger than 128",
			all:     true,
			server:  strings.Repeat("a", 129),
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		ctx := context.Background()
		mockD := newMockDaemon()
		cli := newMockClient(&mockGrpcClient{logoutFunc: mockD.logout})

		logoutOpts.all = tc.all
		logoutOpts.server = tc.server
		_, err := runLogout(ctx, &cli)
		assert.Equal(t, err != nil, tc.wantErr, "Failed at [%s], err: %v", tc.name, err)
		if err != nil {
			assert.ErrorContains(t, err, tc.errString)
		}
	}
}

func TestNewLogoutOptions(t *testing.T) {
	type args struct {
		c    *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		args    args
		flag    string
		wantErr bool
	}{
		{
			name: "TC1 - normal case",
			args: args{
				c:    NewLogoutCmd(),
				args: []string{"test.org"},
			},
			flag:    "--all",
			wantErr: false,
		},
		{
			name: "TC2 - abnormal case with empty server",
			args: args{
				c:    NewLogoutCmd(),
				args: []string{},
			},
			wantErr: true,
		},
		{
			name: "TC3 - abnormal case with more than one args",
			args: args{
				c:    NewLogoutCmd(),
				args: []string{"a", "b"},
			},
			wantErr: true,
		},
		{
			name: "TC4 - abnormal case with invalid server address",
			args: args{
				c:    NewLogoutCmd(),
				args: []string{"/aaaaaa"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.flag != "" {
				tt.args.c.Flag("all").Changed = true
			}
			if err := newLogoutOptions(tt.args.c, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("newLogoutOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewLogoutCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		errString string
	}{
		{
			name:      "TC1 - normal case",
			args:      "test.org",
			errString: "isula_build.sock",
		},
		{
			name:      "TC2 - abnormal case with invalid args",
			args:      "test.org a b",
			errString: "too many arguments",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewLogoutCmd()
			cmd.SetArgs(strings.Split(tt.args, " "))
			err := cmd.Execute()
			if err != nil {

			}
			assert.ErrorContains(t, err, tt.errString)
		})
	}
}
