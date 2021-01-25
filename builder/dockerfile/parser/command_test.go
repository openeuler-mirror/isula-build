// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-03-20
// Description: command parse related tests

package dockerfile

import (
	"testing"

	"gotest.tools/v3/assert"

	"isula.org/isula-build/pkg/parser"
)

func TestParseArg(t *testing.T) {
	type testcase struct {
		str    string
		expect string
		err    string
	}
	var testcases = []testcase{
		{
			str:    "CB_VERSION=6.5.0",
			expect: "(ARG) (CB_VERSION=6.5.0)",
			err:    "",
		},
		{
			str:    "CB_RELEASE_URL=https://packages.couchbase.com/releases/6.5.0",
			expect: "(ARG) (CB_RELEASE_URL=https://packages.couchbase.com/releases/6.5.0)",
			err:    "",
		},
		{
			str:    "CB_PACKAGE=couchbase-server-enterprise_6.5.0-ubuntu16.04_amd64.deb",
			expect: "(ARG) (CB_PACKAGE=couchbase-server-enterprise_6.5.0-ubuntu16.04_amd64.deb)",
			err:    "",
		},
		{
			str:    "CB_SHA256=5505c6bb026090dae7351e9d83caeab00437f19e48e826afd4cb6bafc484cd2b",
			expect: "(ARG) (CB_SHA256=5505c6bb026090dae7351e9d83caeab00437f19e48e826afd4cb6bafc484cd2b)",
			err:    "",
		},
		{
			str:    `USER_HOME_DIR="/root"`,
			expect: `(ARG) (USER_HOME_DIR="/root")`,
			err:    "",
		},
		{
			str:    "BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries",
			expect: "(ARG) (BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries)",
			err:    "",
		},
		{
			str:    "!@#$%^*()-_+foo=isula",
			expect: "(ARG) (!@#$%^*()-_+foo=isula)",
			err:    "",
		},
		{
			str:    "foo =var",
			expect: "",
			err:    "",
		},
		{
			str:    "foo= var",
			expect: "",
			err:    "",
		},
		{
			str:    "foo=",
			expect: "",
			err:    "",
		},
		{
			str:    "foo=var isula",
			expect: "",
			err:    "",
		},
		{
			str:    "!@#$%^*()-_+foo=var isula",
			expect: "",
			err:    "",
		},
	}

	for _, tc := range testcases {
		line := &parser.Line{
			Raw:     tc.str,
			Command: "ARG",
		}
		err := parseArg(line)
		if err != nil {
			assert.ErrorContains(t, err, "format failed")
		} else {
			assert.Equal(t, line.Dump(), tc.expect)
		}
	}
}

func TestParseKeyValue(t *testing.T) {
	type testcase struct {
		str    string
		expect string
		isErr  bool
	}
	var testcases = []testcase{
		{
			str:    "MAVEN_HOME /usr/share/maven",
			expect: `(ARG) (MAVEN_HOME="/usr/share/maven")`,
		},
		{
			str:    "MAVEN_HOME /usr/share/maven /usr/local/maven",
			expect: `(ARG) (MAVEN_HOME="/usr/share/maven /usr/local/maven")`,
		},
		{
			str:    "MAVEN_HOME ",
			expect: "",
			isErr:  true,
		},
		{
			str:    "MAVEN_HOME=",
			expect: "(ARG) (MAVEN_HOME=)",
		},
		{
			str:    "MAVEN_HOME=abc",
			expect: "(ARG) (MAVEN_HOME=abc)",
		},
		{
			str:    "MAVEN_HOME = abc",
			expect: `(ARG) (MAVEN_HOME="= abc")`,
		},
		{
			str:    "MAVEN_HOME = abc GOPATH=~/go",
			expect: `(ARG) (MAVEN_HOME="= abc GOPATH=~/go")`,
		},
		{
			str:    `MAVEN_HOME = abc GOPATH="~/go"`,
			expect: `(ARG) (MAVEN_HOME="= abc GOPATH=\"~/go\"")`,
		},
		{
			str:    `MAVEN_HOME = abc VERSION="robin\"jack ma"`,
			expect: `(ARG) (MAVEN_HOME="= abc VERSION=\"robin\\\"jack ma\"")`,
		},
		{
			str:    `MAVEN_HOME = abc VERSION=" isula isula      isula"`,
			expect: `(ARG) (MAVEN_HOME="= abc VERSION=\" isula isula      isula\"")`,
		},
		{
			str:    `MAVEN_CONFIG "$USER_HOME_DIR/.m2"`,
			expect: `(ARG) (MAVEN_CONFIG="\"$USER_HOME_DIR/.m2\"")`,
		},
		{
			str:    "CONSUL_VERSION=1.7.2",
			expect: "(ARG) (CONSUL_VERSION=1.7.2)",
		},
		{
			str:    "!@#$%^&*()-—+=1.7.2",
			expect: "(ARG) (!@#$%^&*()-—+=1.7.2)",
		},
		{
			str:    "CONSUL_VERSION=1.7.2 DOCKER_VERSION=  18.9.1.100",
			expect: "(ARG) (CONSUL_VERSION=1.7.2) (DOCKER_VERSION=18.9.1.100)",
			isErr:  true,
		},
		{
			str:    "HASHICORP_RELEASES=https://releases.hashicorp.com",
			expect: "(ARG) (HASHICORP_RELEASES=https://releases.hashicorp.com)",
		},
		{
			str:    "PATH=$PATH:/opt/couchbase/bin:/opt/couchbase/bin/tools:/opt/couchbase/bin/install",
			expect: "(ARG) (PATH=$PATH:/opt/couchbase/bin:/opt/couchbase/bin/tools:/opt/couchbase/bin/install)",
		},
		{
			str:    "BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries",
			expect: "(ARG) (BASE_URL=https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries)",
		},
		{
			str:    "CATALINA_HOME /usr/local/tomcat",
			expect: `(ARG) (CATALINA_HOME="/usr/local/tomcat")`,
		},
		{
			str:    "PATH $CATALINA_HOME/bin:$PATH",
			expect: `(ARG) (PATH="$CATALINA_HOME/bin:$PATH")`,
		},
		{
			str:    "TOMCAT_NATIVE_LIBDIR $CATALINA_HOME/native-jni-lib",
			expect: `(ARG) (TOMCAT_NATIVE_LIBDIR="$CATALINA_HOME/native-jni-lib")`,
		},
		{
			str:    "LD_LIBRARY_PATH ${LD_LIBRARY_PATH:+$LD_LIBRARY_PATH:}$TOMCAT_NATIVE_LIBDIR",
			expect: `(ARG) (LD_LIBRARY_PATH="${LD_LIBRARY_PATH:+$LD_LIBRARY_PATH:}$TOMCAT_NATIVE_LIBDIR")`,
		},
		{
			str:    "GPG_KEYS 05AB33110949 07E48665A 4730920",
			expect: `(ARG) (GPG_KEYS="05AB33110949 07E48665A 4730920")`,
		},
		{
			str:    `MAVEN_CONFIG "$USER_HOME_DIR/.m2"`,
			expect: `(ARG) (MAVEN_CONFIG="\"$USER_HOME_DIR/.m2\"")`,
		},
		{
			str:    `testEnv=env1 testEnv2=env2 testEnv3 env3 testEnv4=env4`,
			expect: "",
			isErr:  true,
		},
		{
			str:    `testEnv=env1 testEnv2=env2 testEnv4= env4`,
			expect: "",
			isErr:  true,
		},
		{
			str:    `testEnv= testEnv2=env2`,
			expect: "(ARG) (testEnv=) (testEnv2=env2)",
		},
		{
			str:    `maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"`,
			expect: `(ARG) (maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>")`,
		},
		{
			str:    `arg1 aa`,
			expect: `(ARG) (arg1="aa")`,
		},
		{
			str:    `$arg1 aa`,
			expect: `(ARG) ($arg1="aa")`,
		},
		{
			str:    `$arg-arg aa`,
			expect: `(ARG) ($arg-arg="aa")`,
		},
		{
			str:    `arg.arg aa`,
			expect: `(ARG) (arg.arg="aa")`,
		},
		{
			str:    `1arg.arg aa`,
			expect: `(ARG) (1arg.arg="aa")`,
		},
	}

	for _, tc := range testcases {
		line := &parser.Line{
			Raw:     tc.str,
			Command: "ARG",
		}
		err := parseKeyValue(line)
		if (err != nil) && !tc.isErr {
			t.Errorf("Testing failed. Got: %v, Expected: %v", err, tc.isErr)
		}
		if err == nil {
			assert.Equal(t, line.Dump(), tc.expect)
		}
	}
}

func TestParseAdd(t *testing.T) {
	type testcase struct {
		name        string
		str         string
		isErr       bool
		expectStr   string
		expectFlags map[string]string
	}
	var testcases = []testcase{
		{
			name:        "ParseAdd test 1",
			str:         `--chown=1 ["files*", "/dir/"]`,
			expectFlags: map[string]string{"chown": "1", "attribute": "json"},
			expectStr:   "(ADD) (files*) (/dir/)",
		},
		{
			name:  "ParseAdd test 2",
			str:   `--chown=1 --chown=2 ["files*", "/dir/"]`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 3",
			str:   `--chown=1 ["files*"]`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 4",
			str:   `--chown=1 files*`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 5",
			str:   `--cho=1 ["files*", "/dir/"]`,
			isErr: true,
		},
		{
			name:        "ParseAdd test 6",
			str:         `--chown=1 files* /dir/`,
			expectFlags: map[string]string{"chown": "1"},
			expectStr:   "(ADD) (files*) (/dir/)",
		},
		{
			name:  "ParseAdd test 7",
			str:   `[""]`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 8",
			str:   ``,
			isErr: true,
		},
		{
			name:  "ParseAdd test 9",
			str:   `["files", 123]`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 10",
			str:   `--from=aaa file1 /dir/`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Add,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseAdd(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, line.Dump(), tc.expectStr, "Failed at [%s]", tc.name)
			assert.DeepEqual(t, line.Flags, tc.expectFlags)
		}
	}
}

func TestParseCopy(t *testing.T) {
	type testcase struct {
		name        string
		str         string
		expectStr   string
		expectFlags map[string]string
		isErr       bool
	}
	var testcases = []testcase{
		{
			name:        "ParseCopy test 1",
			str:         `--chown=1 --from=foo ["files*", "/dir/"]`,
			expectStr:   "(COPY) (files*) (/dir/)",
			expectFlags: map[string]string{"chown": "1", "from": "foo", "attribute": "json"},
		},
		{
			name:  "ParseCopy test 2",
			str:   `--chown=1 --chown=2 ["files*", "/dir/"]`,
			isErr: true,
		},
		{
			name:  "ParseCopy test 3",
			str:   `--cho=1 ["files*", "/dir/"]`,
			isErr: true,
		},
		{
			name:        "ParseCopy test 4",
			str:         `--chown=1 --from=foo files* /dir/`,
			expectStr:   "(COPY) (files*) (/dir/)",
			expectFlags: map[string]string{"chown": "1", "from": "foo"},
		},
		{
			name:  "ParseCopy test 5",
			str:   `[""]`,
			isErr: true,
		},
		{
			name:  "ParseCopy test 6",
			str:   ``,
			isErr: true,
		},
		{
			name:  "ParseCopy test 7",
			str:   `--chown=1 ["files*"]`,
			isErr: true,
		},
		{
			name:  "ParseCopy test 8",
			str:   `--chown=1 files*`,
			isErr: true,
		},
		{
			name:  "ParseAdd test 9",
			str:   `["files", 123]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Copy,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseCopy(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, line.Dump(), tc.expectStr, "Failed at [%s]", tc.name)
			assert.DeepEqual(t, line.Flags, tc.expectFlags)
		}
	}
}

func TestParseVolume(t *testing.T) {
	type testcase struct {
		name        string
		str         string
		expectStr   string
		expectFlags map[string]string
		isErr       bool
	}
	var testcases = []testcase{
		{
			name:        "ParseVolume test 1",
			str:         `["/var/lib"]`,
			expectStr:   "(VOLUME) (/var/lib)",
			expectFlags: map[string]string{"attribute": "json"},
		},
		{
			name:  "ParseVolume test 2",
			str:   `--chown=1 ["/var/lib"]`,
			isErr: true,
		},
		{
			name:        "ParseVolume test 3",
			str:         `/var/lib /usr/bin`,
			expectStr:   "(VOLUME) (/var/lib) (/usr/bin)",
			expectFlags: map[string]string{},
		},
		{
			name:  "ParseVolume test 4",
			str:   `[""]`,
			isErr: true,
		},
		{
			name:  "ParseVolume test 5",
			str:   ``,
			isErr: true,
		},
		{
			name:  "ParseVolume test 6",
			str:   `[1]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Volume,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseVolume(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, line.Dump(), tc.expectStr, "Failed at [%s]", tc.name)
			assert.DeepEqual(t, line.Flags, tc.expectFlags)
		}
	}
}

func TestParseCmd(t *testing.T) {
	type testcase struct {
		name   string
		str    string
		expect int
		isErr  bool
	}
	var testcases = []testcase{
		{
			name:   "ParseCmd test 1",
			str:    `/bin/sh -c sleep 1`,
			expect: 1,
		},
		{
			name:   "ParseCmd test 2",
			str:    `["/bin/sh", "-c", "sleep", "1"]`,
			expect: 4,
		},
		{
			name:  "ParseCmd test 3",
			str:   `--chown=1 ["/bin/sh"]`,
			isErr: true,
		},
		{
			name:   "ParseCmd test 4",
			str:    `[""]`,
			expect: 0,
		},
		{
			name:   "ParseCmd test 5",
			str:    ``,
			expect: 0,
		},
		{
			name:   "ParseCmd test 6",
			str:    `["/bin/sh", -c, "sleep", "1"]`,
			expect: 1,
		},
		{
			name:  "ParseCmd test 7",
			str:   `["/bin/sh", "-c", "sleep", 1]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Cmd,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseCmd(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, len(line.Cells), tc.expect, "Failed at [%s]", tc.name)
		}
	}
}

func TestParseEntrypoint(t *testing.T) {
	type testcase struct {
		name   string
		str    string
		expect int
		isErr  bool
	}
	var testcases = []testcase{
		{
			name:   "ParseEntrypoint test 1",
			str:    `/bin/sh -c sleep 1`,
			expect: 1,
		},
		{
			name:   "ParseEntrypoint test 2",
			str:    `["/bin/sh", "-c", "sleep", "1"]`,
			expect: 4,
		},
		{
			name:  "ParseEntrypoint test 3",
			str:   `--chown=1 ["/bin/sh"]`,
			isErr: true,
		},
		{
			name:   "ParseEntrypoint test 4",
			str:    `[""]`,
			expect: 0,
		},
		{
			name:   "ParseEntrypoint test 5",
			str:    ``,
			expect: 0,
		},
		{
			name:   "ParseEntrypoint test 6",
			str:    `["/bin/sh", -c, "sleep", "1"]`,
			expect: 1,
		},
		{
			name:  "ParseEntrypoint test 7",
			str:   `["/bin/sh", "-c", "sleep", 1]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Entrypoint,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseEntrypoint(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, len(line.Cells), tc.expect, "Failed at [%s]", tc.name)
		}
	}
}

func TestParseRun(t *testing.T) {
	type testcase struct {
		name   string
		str    string
		expect int
		isErr  bool
	}
	var testcases = []testcase{
		{
			name:   "ParseRun test 1",
			str:    `/bin/sh -c sleep 1`,
			expect: 1,
		},
		{
			name:   "ParseRun test 2",
			str:    `["/bin/sh", "-c", "sleep", "1"]`,
			expect: 4,
		},
		{
			name:  "ParseRun test 3",
			str:   `--chown=1 ["/bin/sh"]`,
			isErr: true,
		},
		{
			name:   "ParseRun test 4",
			str:    `[""]`,
			expect: 0,
		},
		{
			name:   "ParseRun test 5",
			str:    ``,
			expect: 0,
		},
		{
			name:   "ParseRun test 6",
			str:    `["/bin/sh", -c, "sleep", "1"]`,
			expect: 1,
		},
		{
			name:  "ParseRun test 7",
			str:   `["/bin/sh", "-c", "sleep", 1]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Run,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseRun(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, len(line.Cells), tc.expect, "Failed at [%s]", tc.name)
		}
	}
}

func TestParseShell(t *testing.T) {
	type testcase struct {
		name   string
		str    string
		expect int
		isErr  bool
	}
	var testcases = []testcase{
		{
			name:  "ParseShell test 1",
			str:   `/bin/sh`,
			isErr: true,
		},
		{
			name:   "ParseShell test 2",
			str:    `["powershell", "-command"]`,
			expect: 2,
		},
		{
			name:  "ParseShell test 3",
			str:   ``,
			isErr: true,
		},
		{
			name:  "ParseShell test 4",
			str:   `[""]`,
			isErr: true,
		},
		{
			name:  "ParseShell test 5",
			str:   `[1]`,
			isErr: true,
		},
	}
	for _, tc := range testcases {
		line := &parser.Line{
			Command: Shell,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseShell(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s], err: %v", tc.name, err)
		if err == nil {
			assert.Equal(t, len(line.Cells), tc.expect, "Failed at [%s]", tc.name)
		}
	}
}

func TestParseMaybeString(t *testing.T) {
	type testcase struct {
		str    string
		cmd    string
		expect int
		isErr  bool
		name   string
	}

	var testcases = []testcase{
		{
			name:   "FROM test 1",
			cmd:    From,
			str:    "alpine AS uuid",
			expect: 3,
		},
		{
			name:   "FROM test 2",
			cmd:    From,
			str:    "ubuntu:latest",
			expect: 1,
		},
		{
			name:   "FROM test 3",
			cmd:    From,
			str:    "ubuntu:latest asdf",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "FROM test 4",
			cmd:    From,
			str:    "ubuntu:latest       ",
			expect: 1,
		},
		{
			name:   "FROM test 5",
			cmd:    From,
			str:    "",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "EXPOSE test 1",
			cmd:    Expose,
			str:    "80",
			expect: 1,
		},
		{
			name:   "EXPOSE test 2",
			cmd:    Expose,
			str:    "80/tcp",
			expect: 1,
		},
		{
			name:   "EXPOSE test 3",
			cmd:    Expose,
			str:    "80/tcp 80/udp 8080 3000 5000",
			expect: 5,
		},
		{
			name:   "EXPOSE test 4",
			cmd:    Expose,
			str:    "",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "MAINTAINER test 1",
			cmd:    Maintainer,
			str:    "abcdefghigklmn",
			expect: 1,
		},
		{
			name:   "MAINTAINER test 2",
			cmd:    Maintainer,
			str:    "isula isula-build@isula.com",
			expect: 1,
		},
		{
			name:   "MAINTAINER test 3",
			cmd:    Maintainer,
			str:    "iSula Team <isula-build@isula.org>",
			expect: 1,
		},
		{
			name:   "MAINTAINER test 4",
			cmd:    Maintainer,
			str:    "",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "STOPSIGNAL test 1",
			cmd:    StopSignal,
			str:    "9",
			expect: 1,
		},
		{
			name:   "STOPSIGNAL test 2",
			cmd:    StopSignal,
			str:    "kill",
			expect: 1,
		},
		{
			name:   "STOPSIGNAL test 3",
			cmd:    StopSignal,
			str:    "SIGTERM SIGUSR1 18",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "STOPSIGNAL test 4",
			cmd:    StopSignal,
			str:    "",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "WORKDIR test 1",
			cmd:    WorkDir,
			str:    "/a",
			expect: 1,
		},
		{
			name:   "WORKDIR test 2",
			cmd:    WorkDir,
			str:    "/root abcdefghigklmn",
			expect: 1,
		},
		{
			name:   "WORKDIR test 3",
			cmd:    WorkDir,
			str:    "../mydir",
			expect: 1,
		},
		{
			name:   "WORKDIR test 4",
			cmd:    WorkDir,
			str:    "",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "WORKDIR test 5",
			cmd:    WorkDir,
			str:    "/tmp/test dir/",
			expect: 1,
		},
		{
			name:   "USER test 1",
			cmd:    User,
			str:    "root",
			expect: 1,
		},
		{
			name:   "USER test 2",
			cmd:    User,
			str:    "1000:1000",
			expect: 1,
		},
		{
			name:   "USER test 3",
			cmd:    User,
			str:    "1000 root",
			expect: 1,
			isErr:  true,
		},
		{
			name:   "USER test 4",
			cmd:    User,
			str:    "",
			expect: 1,
			isErr:  true,
		},
	}

	for _, tc := range testcases {
		line := &parser.Line{
			Command: tc.cmd,
			Raw:     tc.str,
			Flags:   make(map[string]string),
		}
		err := parseMaybeString(line)
		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s]", tc.name)
		if err == nil {
			assert.Equal(t, len(line.Cells), tc.expect, "Failed at [%s]", tc.name)
		}
	}
}

func TestParseOnBuild(t *testing.T) {
	type testcase struct {
		str    string
		err    string
		expect string
	}
	var testcases = []testcase{
		{
			str:    "ADD . /app/src",
			expect: "(ONBUILD) (ADD) (.) (/app/src)",
		},
		{
			str:    "RUN /usr/local/bin/python-build --dir /app/src",
			expect: "(ONBUILD) (RUN) (/usr/local/bin/python-build --dir /app/src)",
		},
		{
			str: "",
			err: "requires at least one argument",
		},
		{
			str: "ONBUILD ADD . /app/src",
			err: "isn't allowed as an ONBUILD trigger",
		},
		{
			str: "FROM busybox:latest",
			err: "isn't allowed as an ONBUILD trigger",
		},
		{
			str: "MAINTAINER foo@isula.com",
			err: "isn't allowed as an ONBUILD trigger",
		},
		{
			str: "isula foo@isula.com",
			err: "isn't support",
		},
	}

	for _, tc := range testcases {
		line := &parser.Line{
			Raw:     tc.str,
			Command: "OnBuild",
			Flags:   make(map[string]string),
		}
		err := parseOnBuild(line)
		assert.Equal(t, err != nil, tc.err != "")
		if err == nil {
			assert.Equal(t, line.Dump(), tc.expect)
		} else {
			assert.ErrorContains(t, err, tc.err)
		}
	}
}

func TestParseHealthCheck(t *testing.T) {
	type testcase struct {
		name        string
		str         string
		expectStr   string
		healthFlags map[string]string
		isErr       bool
		errStr      string
	}

	var testcases = []testcase{
		{
			name:   "healthcheck with empty content",
			str:    ` `,
			isErr:  true,
			errStr: "unknown type for healthcheck, need cmd or none",
		},
		{
			name:   "cmd with no command",
			str:    "CMD",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:   "none with command",
			str:    "NONE isula",
			isErr:  true,
			errStr: "none should not take arguments behind it",
		},
		{
			name:   "unknown type",
			str:    "isula",
			isErr:  true,
			errStr: "unknown type for healthcheck, need cmd or none",
		},
		{
			name:        "healthcheck with json cmd",
			str:         `--timeout=3s CMD [ "sleep" ,  "1" ]`,
			expectStr:   "(HEALTHCHECK) (CMD) (sleep) (1)",
			healthFlags: map[string]string{"attribute": "json", "timeout": "3s"},
		},
		{
			name:        "healthcheck with multi flag",
			str:         `--timeout=3s --retries=1 --interval=1s CMD [ "sleep" ,  "1" ]`,
			expectStr:   "(HEALTHCHECK) (CMD) (sleep) (1)",
			healthFlags: map[string]string{"attribute": "json", "timeout": "3s", "retries": "1", "interval": "1s"},
		},
		{
			name:      "healthcheck with invalid retries",
			str:       `--timeout=3s --retries=0 --interval=1s CMD [ "sleep" ,  "1" ]`,
			expectStr: "(HEALTHCHECK) (CMD) (sleep) (1)",
			isErr:     true,
			errStr:    "healthcheck retries must be at least 1",
		},
		{
			name:        "healthcheck with plain cmd",
			str:         "--timeout=3s Cmd sleep 5",
			expectStr:   "(HEALTHCHECK) (CMD) (sleep 5)",
			healthFlags: map[string]string{"timeout": "3s"},
		},
		{
			name:   "unknown flag",
			str:    "  --timeout=3s --a --b cmd sleep 5",
			isErr:  true,
			errStr: "should has specified value with '='",
		},
		{
			name:   "test 5",
			str:    "  --timeout=3s cmd",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:   "test 6",
			str:    "  --timeout=3s cmd   ",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:   "test 7",
			str:    "cmd",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:   "test 8",
			str:    " cmd ",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:   "test 9",
			str:    " cmd ",
			isErr:  true,
			errStr: "missing command after healthcheck cmd",
		},
		{
			name:        "test 10",
			str:         "None",
			expectStr:   "(HEALTHCHECK) (NONE)",
			healthFlags: map[string]string{},
		},
		{
			name:        "test 11",
			str:         "none",
			expectStr:   "(HEALTHCHECK) (NONE)",
			healthFlags: map[string]string{},
		},
		{
			name:        "test 12",
			str:         "   none  ",
			expectStr:   "(HEALTHCHECK) (NONE)",
			healthFlags: map[string]string{},
		},
		{
			name:   "test 13",
			str:    "   none  cmd sleep",
			isErr:  true,
			errStr: "none should not take arguments behind it",
		},
		{
			name:        "test 14",
			str:         `--timeout=3ssss CMD [ "sleep"]`,
			expectStr:   "(HEALTHCHECK) (CMD) (sleep)",
			healthFlags: map[string]string{"timeout": "3ssss", "attribute": "json"},
		},
		{
			name:   "test 15",
			str:    `--ttttout=3ssss CMD [ "sleep"    ]`,
			isErr:  true,
			errStr: "unknown flag",
		},
		{
			name:      "cmd json with int",
			str:       `CMD [ "slllllllleep",1]`,
			expectStr: `(HEALTHCHECK) (CMD) (slllllllleep) (1)`,
			isErr:     true,
			errStr:    "only string type is allowd as JSON format arrays",
		},
		{
			name:      "cmd json with float",
			str:       `CMD [ "slllllllleep", 1.5]`,
			expectStr: `(HEALTHCHECK) (CMD) (slllllllleep) (1.5)`,
			isErr:     true,
			errStr:    "only string type is allowd as JSON format arrays",
		},
		{
			name:        "invalid cmd json with mix of digit and character",
			str:         `CMD [ "slllllllleep", abc1.5efg12]`,
			expectStr:   `(HEALTHCHECK) (CMD) ([ "slllllllleep", abc1.5efg12])`,
			healthFlags: map[string]string{},
		},
		{
			name:   "test 17",
			str:    `none cmd ok [ "slllllllleep",1]`,
			isErr:  true,
			errStr: "none should not take arguments behind it",
		},
		{
			name:   "test 17",
			str:    `none --a --b --c cmd ok [ "slllllllleep",1]`,
			isErr:  true,
			errStr: "none should not take arguments behind it",
		},
		{
			name:   "test 18",
			str:    `none --a --b --c cmd ok [[[[]`,
			isErr:  true,
			errStr: "none should not take arguments behind it",
		},
		{
			name:   "test 19",
			str:    `www --a --b --c cmd ok [[[[]`,
			isErr:  true,
			errStr: "wrong argument before CMD or NONE",
		},
		{
			name:   "test 20",
			str:    `--1 --2 --3 cmd --a --b cmd ok [[[[]`,
			isErr:  true,
			errStr: "should has specified value with '='",
		},
		{
			name:   "test 21",
			str:    `--w cmd --a --b `,
			isErr:  true,
			errStr: "should has specified value with '='",
		},
	}

	for _, tc := range testcases {
		line := &parser.Line{
			Raw:     tc.str,
			Command: "HEALTHCHECK",
			Flags:   make(map[string]string),
		}
		err := parseHealthCheck(line)

		assert.Equal(t, err != nil, tc.isErr, "Failed at [%s]", tc.name)
		if err != nil {
			assert.ErrorContains(t, err, tc.errStr)
		} else {
			assert.Equal(t, line.Dump(), tc.expectStr, "Failed at [%s]", tc.name)
			assert.DeepEqual(t, line.Flags, tc.healthFlags)
		}
	}
}
