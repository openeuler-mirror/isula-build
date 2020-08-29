// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jingxiao Lu
// Create: 2020-03-20
// Description: dockerfile parse common functions tests

package dockerfile

import (
	"testing"
)

func TestRegParam(t *testing.T) {
	var tests = []struct {
		name      string
		param     string
		matchFull bool
	}{
		{
			name:      "name 1",
			param:     "testArg",
			matchFull: true,
		},
		{
			name:      "name 2",
			param:     "test_arg",
			matchFull: true,
		},
		{
			name:      "name 3",
			param:     "testArg2",
			matchFull: true,
		},
		{
			name:      "name 4",
			param:     "test-Arg2",
			matchFull: true,
		},
		{
			name:      "name 10",
			param:     "1testArg",
			matchFull: false,
		},
		{
			name:      "name 11",
			param:     "test*Arg",
			matchFull: false,
		},
		{
			name:      "name 12",
			param:     "test&Arg",
			matchFull: false,
		},
		{
			name:      "name 13",
			param:     "test###Arg",
			matchFull: false,
		},
		{
			name:      "name 14",
			param:     "test(Arg)2",
			matchFull: false,
		},
		{
			name:      "name 15",
			param:     "1212",
			matchFull: false,
		},
		{
			name:      "name 16",
			param:     "A",
			matchFull: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subs := regParam.FindAllString(tt.param, -1)
			if tt.matchFull {
				if len(subs) != 1 {
					t.FailNow()
				}
				if len(subs[0]) != len(tt.param) {
					t.FailNow()
				}
			} else if len(subs) == 1 && len(subs[0]) == len(tt.param) {
				t.FailNow()
			}
		})
	}
}

func TestRegParamSpecial(t *testing.T) {
	var tests = []struct {
		name      string
		param     string
		matchFull bool
	}{
		{
			name:      "name 1",
			param:     "testArg",
			matchFull: true,
		},
		{
			name:      "name 2",
			param:     "test_arg",
			matchFull: true,
		},
		{
			name:      "name 3",
			param:     "test-arg",
			matchFull: true,
		},
		{
			name:      "name 4",
			param:     "test:-arg",
			matchFull: true,
		},
		{
			name:      "name 5",
			param:     "test:+arg",
			matchFull: true,
		},
		{
			name:      "name 11",
			param:     "test:=arg",
			matchFull: false,
		},
		{
			name:      "name 12",
			param:     "test:arg",
			matchFull: false,
		},
		{
			name:      "name 13",
			param:     "test:?arg",
			matchFull: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subs := regParamSpecial.FindAllString(tt.param, -1)
			if tt.matchFull {
				if len(subs) != 1 {
					t.FailNow()
				}
				if len(subs[0]) != len(tt.param) {
					t.FailNow()
				}
			} else if len(subs) == 1 && len(subs[0]) != len(tt.param) {
				t.FailNow()
			}
		})
	}
}

func TestResolveParam(t *testing.T) {
	type args struct {
		s          string
		strict     bool
		resolveArg func(string) string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "case 001 - no param",
			args: args{
				s:          "testParam",
				strict:     false,
				resolveArg: nil,
			},
			want:    "testParam",
			wantErr: false,
		},
		{
			name: "case 002 - empty",
			args: args{
				s:          "$",
				strict:     false,
				resolveArg: nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case 003 - empty",
			args: args{
				s:          "${}",
				strict:     false,
				resolveArg: nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case 004 - no matching }",
			args: args{
				s:          "${testArg",
				strict:     false,
				resolveArg: func(s string) string { return "yes" },
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case 005 - no valid arg",
			args: args{
				s:          "$=$",
				strict:     false,
				resolveArg: nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case 006 - special chars",
			args: args{
				s:          "$=#%",
				strict:     false,
				resolveArg: nil,
			},
			want:    "#%",
			wantErr: false,
		},
		{
			// the '#' after '$' will be converted and ignored
			name: "case 007 - special chars",
			args: args{
				s:          "$#v=*",
				strict:     false,
				resolveArg: nil,
			},
			want:    "v=*",
			wantErr: false,
		},
		{
			// the '%' after '$' will be converted and ignored
			name: "case 008 - special chars",
			args: args{
				s:          "#*=@#$%",
				strict:     false,
				resolveArg: nil,
			},
			want:    "#*=@#",
			wantErr: false,
		},
		{
			// the '%' after '$' will be converted and ignored
			name: "case 009 - special chars",
			args: args{
				s:          "${*=}",
				strict:     false,
				resolveArg: nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case 101",
			args: args{
				s:      "${testArg}",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "found",
			wantErr: false,
		},
		{
			name: "case 102",
			args: args{
				s:      "${testArg}foo",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "foundfoo",
			wantErr: false,
		},
		{
			name: "case 103",
			args: args{
				s:      "$testArg:tag",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "found:tag",
			wantErr: false,
		},
		{
			name: "case 104",
			args: args{
				s:      `\$testArg:tag`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    `\$testArg:tag`,
			wantErr: false,
		},
		{
			name: "case 105",
			args: args{
				s:      `$testArg:\$tag`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    `found:\$tag`,
			wantErr: false,
		},
		{
			name: "case 106",
			args: args{
				s:      `\$`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    `\$`,
			wantErr: false,
		},
		{
			name: "case 107",
			args: args{
				s:      `${A}`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "A" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    `found`,
			wantErr: false,
		},
		{
			// ${variable:-name}, match variable, return $variable
			name: "case 201",
			args: args{
				s:      "${testArg:-word}",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg:-word" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "found",
			wantErr: false,
		},
		{
			// ${variable:+name}, match variable, return name
			name: "case 202",
			args: args{
				s:      "${testArg:+word}",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg:+word" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "found",
			wantErr: false,
		},
		{
			// strict mode test for FROM command. No matching for name1, return "", then got err
			name: "case 301",
			args: args{
				s:      "${name1}",
				strict: true,
				resolveArg: func(s string) string {
					if s == "name1" {
						return ""
					} else {
						return "busybox"
					}
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			// easy mode (v.s. strict) test for other commands. No matching for name1, return "", no err just ignore "name1"
			name: "case 302",
			args: args{
				s:      "${name1}*Arg",
				strict: false,
				resolveArg: func(s string) string {
					if s == "name1" {
						return ""
					} else {
						return "busybox"
					}
				},
			},
			want:    "*Arg",
			wantErr: false,
		},
		{
			// strict mode test for FROM command. No matching for name1, return "", then got err
			name: "case 303",
			args: args{
				s:      "$name1*Arg",
				strict: true,
				resolveArg: func(s string) string {
					if s == "name1" {
						return ""
					} else {
						return "busybox"
					}
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			// easy mode (v.s. strict) test for other commands. No matching for name1, return "", no err just ignore "name1"
			name: "case 304",
			args: args{
				s:      "$name1*Arg",
				strict: false,
				resolveArg: func(s string) string {
					if s == "name1" {
						return ""
					} else {
						return "busybox"
					}
				},
			},
			want:    "*Arg",
			wantErr: false,
		},
		{
			// strict mode test for FROM command. Special chars after '$', return error
			name: "case 305",
			args: args{
				s:          "$=*",
				strict:     true,
				resolveArg: nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			// easy mode (v.s. strict) test for other commands. Special chars after '$', ignore
			name: "case 306",
			args: args{
				s:          "$=*",
				strict:     false,
				resolveArg: nil,
			},
			want:    "*",
			wantErr: false,
		},
		{
			// single quotes
			name: "case 401 - quotes",
			args: args{
				s:      "'testArg'foo",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "testArgfoo",
			wantErr: false,
		},
		{
			// single quotes
			name: "case 402 - quotes",
			args: args{
				s:      "'testArgfoo",
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			// double quotes
			name: "case 403 - quotes",
			args: args{
				s:      `"testArg"foo`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "testArgfoo",
			wantErr: false,
		},
		{
			// double quotes
			name: "case 404 - quotes",
			args: args{
				s:      `"testArgfoo`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			// fixed quotes
			name: "case 405 - quotes",
			args: args{
				s:      `"testArg'foo`,
				strict: false,
				resolveArg: func(s string) string {
					if s == "testArg" {
						return "found"
					} else {
						return "fail"
					}
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveParam(tt.args.s, tt.args.strict, tt.args.resolveArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveParam() got = %v, want %v", got, tt.want)
			}
		})
	}
}
