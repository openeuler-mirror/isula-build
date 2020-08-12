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
// Create: 2020-01-20
// Description: This file is used for client testing

package main

import (
	"testing"
	"time"
)

func TestGetStartTimeout(t *testing.T) {
	type args struct {
		timeout string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "TC1 - normal case",
			args:    args{timeout: "1s"},
			want:    time.Second,
			wantErr: false,
		},
		{
			name:    "TC2 - normal case with empty timeout input",
			args:    args{timeout: ""},
			want:    defaultStartTimeout,
			wantErr: false,
		},
		{
			name:    "TC3 - abnormal case with larger than max start timeout",
			args:    args{timeout: "21s"},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "TC4 - abnormal case with less than min start timeout",
			args:    args{timeout: "19ms"},
			want:    -1,
			wantErr: true,
		},
		{
			name:    "TC5 - abnormal case with invalid timeout format",
			args:    args{timeout: "abc"},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getStartTimeout(tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("getStartTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getStartTimeout() got = %v, want %v", got, tt.want)
			}
		})
	}
}
