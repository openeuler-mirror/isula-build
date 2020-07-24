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
// Create: 2020-07-20
// Description: This file is used for testing port setting

package util

import (
	"testing"
)

func TestPortSet(t *testing.T) {
	type args struct {
		rawPort string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "TC1 - normal case",
			args:    args{rawPort: "8080"},
			want:    "8080/tcp",
			wantErr: false,
		},
		{
			name:    "TC2 - normal case",
			args:    args{rawPort: "8080/"},
			want:    "8080/tcp",
			wantErr: false,
		},
		{
			name:    "TC3 - abnormal case with invalid port format",
			args:    args{rawPort: "aaa"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "TC4 - normal case with port range",
			args:    args{rawPort: "3000-5000/udp"},
			want:    "3000-5000/udp",
			wantErr: false,
		},
		{
			name:    "TC5 - abnormal case with invalid port range",
			args:    args{rawPort: "3000-500/udp"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "TC6 - abnormal case with invalid port range number",
			args:    args{rawPort: "a-b/udp"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "TC7 - abnormal case with invalid port range numbers",
			args:    args{rawPort: "3000-5000-8000/udp"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "TC8 - abnormal case with empty port",
			args:    args{rawPort: ""},
			want:    "",
			wantErr: true,
		},
		{
			name:    "TC9 - abnormal case with invalid protocol",
			args:    args{rawPort: "80/abc"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PortSet(tt.args.rawPort)
			if (err != nil) != tt.wantErr {
				t.Errorf("PortSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PortSet() got = %v, want %v", got, tt.want)
			}
		})
	}
}
