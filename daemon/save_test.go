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
// Create: 2020-08-19
// Description: This is test file for save

package daemon

import (
	"testing"
)

func TestCheckTag(t *testing.T) {
	type args struct {
		oriImg  string
		imageID string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TC1 - normal case use imageID",
			args: args{
				oriImg:  "123",
				imageID: "123456",
			},
			want: "",
			wantErr: false,
		},
		{
			name: "TC2 - normal case has no input",
			args: args{
				oriImg:  "",
				imageID: "",
			},
			want: "",
			wantErr: false,
		},
		{
			name: "TC3 - normal case use name",
			args: args{
				oriImg:  "tag123",
				imageID: "tag12456",
			},
			want: "tag123:latest",
			wantErr: false,
		},
		{
			name: "TC4 - normal case with tag",
			args: args{
				oriImg:  "tag124:v1",
				imageID: "tag12456",
			},
			want: "tag124:v1",
			wantErr: false,
		},
		{
			name: "TC5 - abnormal case with multiple tags",
			args: args{
				oriImg:  "localhost:5000:5000/isula:latest",
				imageID: "91e6c776a1dccde22f4f90dbee5a2c0a",
			},
			want: "",
			wantErr:true,
		},
		{
			name: "TC6 - abnormal case with invalid tag",
			args: args{
				oriImg:  "isula!@#:latest",
				imageID: "91e6c776a1dccde22f4f90dbee5a2c0a",
			},
			want: "",
			wantErr:true,
		},
		{
			name: "TC7 - abnormal case with invalid tag 2",
			args: args{
				oriImg:  "isula :latest",
				imageID: "91e6c776a1dccde22f4f90dbee5a2c0a",
			},
			want: "",
			wantErr:true,
		},
		{
			name: "TC8 - abnormal case with invalid tag 3",
			args: args{
				oriImg:  " isula:latest",
				imageID: "91e6c776a1dccde22f4f90dbee5a2c0a",
			},
			want: "",
			wantErr:true,
		},
		{
			name: "TC9 - abnormal case with invalid tag 4",
			args: args{
				oriImg:  "isula:latest ",
				imageID: "91e6c776a1dccde22f4f90dbee5a2c0a",
			},
			want: "",
			wantErr:true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkTag(tt.args.oriImg, tt.args.imageID)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkTag() got = %v, want %v", got, tt.want)
			}
		})
	}
}
