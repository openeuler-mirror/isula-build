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
// Description: Builder related functions tests

package dockerfile

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
	testutil "isula.org/isula-build/util"
)

func TestParseFiles(t *testing.T) {
	dockerfile := `
ARG testArg
FROM alpine AS uuid
COPY uuid /src/data

FROM alpine AS date
COPY date /src/data

FROM alpine
COPY --from=uuid /src/data /uuid
COPY --from=date /src/data /date
`
	dockerignore := `
# comment
*/temp*
*/*/temp*
temp?`
	ctxDir := fs.NewDir(t, t.Name(), fs.WithFile(".dockerignore", dockerignore))
	defer ctxDir.Remove()

	b := &Builder{
		buildOpts: BuildOptions{
			ContextDir: ctxDir.Path(),
			File:       dockerfile,
		},
	}

	err := b.parseFiles()
	assert.NilError(t, err)

	expectedPlayBook := &parser.PlayBook{
		HeadingArgs: []string{"testArg"},
		Pages: []*parser.Page{
			{
				Name: "uuid",
				Lines: []*parser.Line{
					{
						Cells: []*parser.Cell{
							{Value: "alpine"},
							{Value: "AS"},
							{Value: "uuid"},
						},
						Begin:   3,
						End:     3,
						Command: "FROM",
						Raw:     "alpine AS uuid",
						Flags:   make(map[string]string),
					},
					{
						Cells: []*parser.Cell{
							{Value: "uuid"},
							{Value: "/src/data"},
						},
						Begin:   4,
						End:     4,
						Command: "COPY",
						Raw:     "uuid /src/data",
						Flags:   make(map[string]string),
					},
				},
				Begin: 3,
				End:   4,
			},
			{
				Name: "date",
				Lines: []*parser.Line{
					{
						Cells: []*parser.Cell{
							{Value: "alpine"},
							{Value: "AS"},
							{Value: "date"},
						},
						Begin:   6,
						End:     6,
						Command: "FROM",
						Raw:     "alpine AS date",
						Flags:   make(map[string]string),
					},
					{
						Cells: []*parser.Cell{
							{Value: "date"},
							{Value: "/src/data"},
						},
						Begin:   7,
						End:     7,
						Command: "COPY",
						Raw:     "date /src/data",
						Flags:   make(map[string]string),
					},
				},
				Begin: 6,
				End:   7,
			},
			{
				Name: "2",
				Lines: []*parser.Line{
					{
						Cells: []*parser.Cell{
							{Value: "alpine"},
						},
						Begin:   9,
						End:     9,
						Command: "FROM",
						Raw:     "alpine",
						Flags:   make(map[string]string),
					},
					{
						Cells: []*parser.Cell{
							{Value: "/src/data"},
							{Value: "/uuid"},
						},
						Begin:   10,
						End:     10,
						Command: "COPY",
						Raw:     "--from=uuid /src/data /uuid",
						Flags:   map[string]string{"from": "uuid"},
					},
					{
						Cells: []*parser.Cell{
							{Value: "/src/data"},
							{Value: "/date"},
						},
						Begin:   11,
						End:     11,
						Command: "COPY",
						Raw:     "--from=date /src/data /date",
						Flags:   map[string]string{"from": "date"},
					},
				},
				Begin:      9,
				NeedCommit: true,
				End:        11,
			},
		},
		Warnings: nil,
	}
	assert.DeepEqual(t, b.playbook, expectedPlayBook)

	expectedIgnores := strings.Split(dockerignore, "\n")
	// expectedIgnores[2:0]: trim empty line and comment line
	assert.DeepEqual(t, b.ignores, expectedIgnores[2:])
}

func TestAnalysePlayBookWithStageName(t *testing.T) {
	dockerfile := `
FROM alpine AS noarg
RUN ls

FROM busybox AS hasarg
RUN ls

FROM alpine
RUN ls

FROM noarg
RUN ls
`
	b := &Builder{
		buildOpts: BuildOptions{
			File: dockerfile,
		},
	}
	err := b.parseFiles()
	assert.NilError(t, err)
	err = b.newStageBuilders()
	assert.NilError(t, err)

	// check the arg and env taken by the command: RUN ls
	assert.DeepEqual(t, b.stageBuilders[0].fromImage, "alpine")
	assert.DeepEqual(t, b.stageBuilders[0].name, "noarg")
	assert.DeepEqual(t, b.stageBuilders[1].fromImage, "busybox")
	assert.DeepEqual(t, b.stageBuilders[1].name, "hasarg")
	assert.DeepEqual(t, b.stageBuilders[2].fromImage, "alpine")
	assert.DeepEqual(t, b.stageBuilders[2].name, "2")
	assert.DeepEqual(t, b.stageBuilders[3].fromImage, "noarg")
	assert.DeepEqual(t, b.stageBuilders[3].name, "3")
}

func TestAnalysePlayBookWithNoArgBeforeFrom(t *testing.T) {
	dockerfile := `
FROM alpine AS noArg
RUN ls

FROM alpine AS hasArg
ARG testArg
RUN ls

FROM alpine AS hasSameEnv
ARG testArg
ENV testArg 1.0
RUN ls
`
	b := &Builder{
		buildOpts: BuildOptions{
			File:      dockerfile,
			BuildArgs: map[string]string{"testArg": "0.1", "no_proxy": "10.0.0.0"},
		},
		ctx: context.Background(),
	}
	err := b.parseFiles()
	assert.NilError(t, err)
	err = b.newStageBuilders()
	assert.NilError(t, err)
	for _, sb := range b.stageBuilders {
		err = sb.analyzeStage(context.Background())
		assert.NilError(t, err)
	}

	// check the arg and env taken by the command: RUN ls
	assert.DeepEqual(t, b.stageBuilders[0].commands[0].args,
		map[string]string{"no_proxy": "10.0.0.0"})
	assert.DeepEqual(t, b.stageBuilders[1].commands[1].args,
		map[string]string{"testArg": "0.1", "no_proxy": "10.0.0.0"})
	assert.DeepEqual(t, b.stageBuilders[2].commands[2].args,
		map[string]string{"no_proxy": "10.0.0.0"})
	assert.DeepEqual(t, b.stageBuilders[2].commands[2].envs,
		map[string]string{"testArg": "1.0"})
}

func TestAnalysePlayBookWithArgBeforeFrom(t *testing.T) {
	dockerfile := `
ARG testArg=0.1
FROM alpine AS noArg
RUN ls

FROM alpine AS hasArg
ARG testArg
RUN ls

FROM alpine AS hasSameEnv
ARG testArg
ENV testArg 1.0
RUN ls
`
	b := &Builder{
		buildOpts: BuildOptions{
			File:      dockerfile,
			BuildArgs: map[string]string{"HTTPS_PROXY": "127.0.0.1"},
		},
		ctx: context.Background(),
	}

	err := b.parseFiles()
	assert.NilError(t, err)
	err = b.newStageBuilders()
	assert.NilError(t, err)
	for _, sb := range b.stageBuilders {
		err = sb.analyzeStage(context.Background())
		assert.NilError(t, err)
	}

	// check the arg and env taken by the command: RUN ls
	assert.DeepEqual(t, b.stageBuilders[0].commands[0].args,
		map[string]string{"HTTPS_PROXY": "127.0.0.1"})
	assert.DeepEqual(t, b.stageBuilders[1].commands[1].args,
		map[string]string{"testArg": "0.1", "HTTPS_PROXY": "127.0.0.1"})
	assert.DeepEqual(t, b.stageBuilders[2].commands[2].args,
		map[string]string{"HTTPS_PROXY": "127.0.0.1"})
	assert.DeepEqual(t, b.stageBuilders[2].commands[2].envs,
		map[string]string{"testArg": "1.0"})
}

// the tested Dockerfile changes from https://github.com/moby/moby/issues/18119#issuecomment-589704252
func TestAnalysePlayBookWithArgs(t *testing.T) {
	dockerfile := `
ARG GLOBALARG_ONE=globalarg_one_default_value
ARG GLOBALARG_TWO=globalarg_two_default_value

FROM myimage:${GLOBALARG_ONE} AS stage1
ARG STAGE1_ARG=foo
RUN ls
ARG GLOBALARG_ONE
RUN ls
ARG GLOBALARG_TWO=my-local-value
RUN ls

FROM ${GLOBALARG_TWO}:${GLOBALARG_ONE} AS stage1
RUN ls
ARG STAGE2_ARG=bar
RUN ls
`
	b := &Builder{
		buildOpts: BuildOptions{
			File:      dockerfile,
			BuildArgs: map[string]string{"GLOBALARG_TWO": "override"},
		},
		ctx: context.Background(),
	}

	err := b.parseFiles()
	assert.NilError(t, err)
	err = b.newStageBuilders()
	assert.NilError(t, err)
	for _, sb := range b.stageBuilders {
		err = sb.analyzeStage(context.Background())
		assert.NilError(t, err)
	}

	// check the arg and env taken by the command: RUN ls
	assert.DeepEqual(t, b.stageBuilders[0].fromImage, "myimage:globalarg_one_default_value")
	assert.DeepEqual(t, b.stageBuilders[0].commands[1].args, map[string]string{"STAGE1_ARG": "foo"})
	assert.DeepEqual(t, b.stageBuilders[0].commands[3].args,
		map[string]string{"STAGE1_ARG": "foo", "GLOBALARG_ONE": "globalarg_one_default_value"})
	assert.DeepEqual(t, b.stageBuilders[0].commands[5].args,
		map[string]string{"STAGE1_ARG": "foo", "GLOBALARG_ONE": "globalarg_one_default_value", "GLOBALARG_TWO": "override"})

	//assert.DeepEqual(t, b.stageBuilders[1].(*stageBuilder).fromImage, "override:globalarg_one_default_value")
	assert.DeepEqual(t, b.stageBuilders[1].commands[0].args,
		map[string]string{})
	assert.DeepEqual(t, b.stageBuilders[1].commands[2].args,
		map[string]string{"STAGE2_ARG": "bar"})
}

func TestUsedHeadingArgs(t *testing.T) {
	type fields struct {
		buildOpts  BuildOptions
		playbook   *parser.PlayBook
		unusedArgs map[string]string
	}
	type wants struct {
		headingArgs map[string]string
		reserved    map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   wants
	}{
		{
			// no matching args between BuildArgs and HeadingArgs
			name: "no matching args between BuildArgs and HeadingArgs",
			fields: fields{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
				},
				playbook: &parser.PlayBook{
					HeadingArgs: []string{"harg1", "harg2"},
				},
				unusedArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
			},
			want: wants{map[string]string{}, map[string]string{}},
		},
		{
			// no matching args between BuildArgs and HeadingArgs, and 1 HeadingArgs has default value
			name: "no matching args between BuildArgs and HeadingArgs, and 1 HeadingArgs has default value",
			fields: fields{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
				},
				playbook: &parser.PlayBook{
					HeadingArgs: []string{"harg1", "harg2=arg2"},
				},
				unusedArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
			},
			want: wants{map[string]string{"harg2": "arg2"}, map[string]string{}},
		},
		{
			// has 2 matching args
			name: "has 2 matching args, and 1 HeadingArgs has default value",
			fields: fields{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"},
				},
				playbook: &parser.PlayBook{
					HeadingArgs: []string{"barg1", "barg2=arg2", "harg3=hhhharg3"},
				},
				unusedArgs: map[string]string{"barg3": "arg3"},
			},
			want: wants{map[string]string{"barg1": "arg1", "barg2": "arg2", "harg3": "hhhharg3"}, map[string]string{}},
		},
		{
			// all matched
			name: "all matched",
			fields: fields{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"},
				},
				playbook: &parser.PlayBook{
					HeadingArgs: []string{"barg1", "barg2", "barg3=arg3"},
				},
				unusedArgs: make(map[string]string),
			},
			want: wants{map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"}, map[string]string{}},
		},
		{
			// preserved args
			name: "preserved args",
			fields: fields{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{"barg1": "arg1", "HTTP_PROXY": "http://www.test.com/", "no_proxy": "10.0.0.1"},
				},
				playbook: &parser.PlayBook{
					HeadingArgs: []string{"barg1=arg1", "barg2=arg2"},
				},
				unusedArgs: map[string]string{},
			},
			want: wants{map[string]string{"barg1": "arg1", "barg2": "arg2"},
				map[string]string{"HTTP_PROXY": "http://www.test.com/", "no_proxy": "10.0.0.1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				buildOpts:  tt.fields.buildOpts,
				playbook:   tt.fields.playbook,
				unusedArgs: tt.fields.unusedArgs,
			}
			err := b.usedHeadingArgs()
			assert.NilError(t, err)
			if !reflect.DeepEqual(b.headingArgs, tt.want.headingArgs) {
				t.Errorf("usedHeadingArgs() headingArgs = %v, want %v", b.headingArgs, tt.want.headingArgs)
			}
			if !reflect.DeepEqual(b.reservedArgs, tt.want.reserved) {
				t.Errorf("usedHeadingArgs() reserved = %v, want %v", b.reservedArgs, tt.want.reserved)
			}
		})
	}
}

func TestAnalysePlayBookWithUnusedArgs(t *testing.T) {
	var testcases = []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		funcCheck  func(t *testing.T, b *Builder)
	}{
		{
			name: "test 1",
			dockerfile: `ARG testArg
FROM alpine AS stage1
ARG testArg2=foo

FROM stage1
ARG testArg3`,
			buildArgs: map[string]string{"unusedArg1": "arg1", "testArg2": "arg2"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"unusedArg1": "arg1"})
			},
		}, {
			name: "test 2 - no matching args between BuildArgs and HeadingArgs",
			dockerfile: `ARG harg1
ARG harg2
FROM alpine
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"barg1": "arg1", "barg2": "arg2"})
			},
		}, {
			name: "test 3 - no matching args between BuildArgs and HeadingArgs, and 1 HeadingArgs has default value",
			dockerfile: `ARG harg1
ARG harg2=arg2
FROM alpine
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{"harg2": "arg2"})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"barg1": "arg1", "barg2": "arg2"})
			},
		}, {
			name: "test 4 - has 2 matching args, and 1 HeadingArgs has default value",
			dockerfile: `ARG barg1
ARG barg2=arg2
ARG harg3=hhhharg3
FROM alpine
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{"barg1": "arg1", "barg2": "arg2", "harg3": "hhhharg3"})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"barg3": "arg3"})
			},
		}, {
			name: "test 5 - all matched",
			dockerfile: `ARG barg1
ARG barg2
ARG barg3
FROM alpine
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{"barg1": "arg1", "barg2": "arg2", "barg3": "arg3"})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{})
			},
		}, {
			name: "test 6 - preserved args",
			dockerfile: `ARG barg1=arg1
ARG barg2=arg2
ARG barg3
FROM alpine
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1", "HTTP_PROXY": "http://www.test.com/", "no_proxy": "10.0.0.1"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{"barg1": "arg1", "barg2": "arg2"})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{"HTTP_PROXY": "http://www.test.com/", "no_proxy": "10.0.0.1"})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{})
			},
		}, {
			name: "test 7 - FROM args",
			dockerfile: `ARG barg1
FROM ${barg1:+busybox}
WORKDIR /home`,
			buildArgs: map[string]string{"barg1": "arg1"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.headingArgs, map[string]string{"barg1": "arg1"})
				assert.DeepEqual(t, b.reservedArgs, map[string]string{})
				assert.DeepEqual(t, b.unusedArgs, map[string]string{})
				assert.DeepEqual(t, b.stageBuilders[0].fromImage, "busybox")
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				buildOpts: BuildOptions{
					File:      tt.dockerfile,
					BuildArgs: tt.buildArgs,
				},
				ctx: context.Background(),
			}

			err := b.parseFiles()
			assert.NilError(t, err)
			err = b.newStageBuilders()
			assert.NilError(t, err)
			for _, sb := range b.stageBuilders {
				err = sb.analyzeStage(context.Background())
				assert.NilError(t, err)
			}
			tt.funcCheck(t, b)
		})
	}
}

func TestCleanupBuilder(t *testing.T) {
	dockerfile := `
ARG testArg
FROM alpine AS stage1
ARG testArg2=foo

FROM stage1
ARG testArg3
`
	ctx := context.Background()
	b := &Builder{
		buildOpts: BuildOptions{
			File:      dockerfile,
			BuildArgs: map[string]string{"unusedArg1": "arg1", "testArg2": "arg2", "unnamedArg": "arg3"},
		},
		cliLog: logger.NewCliLogger(constant.CliLogBufferLen),
		ctx:    context.WithValue(ctx, util.BuildDirKey(util.BuildDir), "/tmp/isula-build-test"),
	}

	err := b.parseFiles()
	assert.NilError(t, err)
	err = b.newStageBuilders()
	assert.NilError(t, err)
	for _, sb := range b.stageBuilders {
		err = sb.analyzeStage(context.Background())
		assert.NilError(t, err)
	}

	b.cleanup()
	logMsg, ok := <-b.cliLog.GetContent()
	assert.Equal(t, ok, true)
	assert.Equal(t, logMsg, "[Warning] One or more build-args [unnamedArg unusedArg1] were not consumed\n")
}

func TestGetFlagsAndArgs(t *testing.T) {
	type testcase struct {
		line        *parser.Line
		expectArgs  []string
		expectFlags map[string]string
	}

	var testcases = []testcase{
		// 1. all
		{
			line: &parser.Line{
				Command: "HEALTHCHECK",
				Cells: []*parser.Cell{
					{
						Value: "CMD",
					},
					{
						Value: "curl -fs http://localhost/ || exit 1",
					},
				},
				Flags: map[string]string{"start-period": "5s", "interval": "5s", "timeout": "3s", "retries": "3"},
			},
			expectArgs:  []string{"CMD", "curl -fs http://localhost/ || exit 1"},
			expectFlags: map[string]string{"start-period": "5s", "interval": "5s", "timeout": "3s", "retries": "3"},
		},
		// 2. no retries
		{
			line: &parser.Line{
				Command: "HEALTHCHECK",
				Cells: []*parser.Cell{
					{
						Value: "CMD",
					},
					{
						Value: "curl -fs http://localhost/ || exit 1",
					},
				},
				Flags: map[string]string{"start-period": "5s", "interval": "5s", "timeout": "3s"},
			},
			expectArgs:  []string{"CMD", "curl -fs http://localhost/ || exit 1"},
			expectFlags: map[string]string{"start-period": "5s", "interval": "5s", "timeout": "3s"},
		},
		// 3. no timeout
		{
			line: &parser.Line{
				Command: "HEALTHCHECK",
				Cells: []*parser.Cell{
					{
						Value: "CMD",
					},
					{
						Value: "curl -fs http://localhost/ || exit 1",
					},
				},
				Flags: map[string]string{"start-period": "5s", "interval": "5s"},
			},
			expectArgs:  []string{"CMD", "curl -fs http://localhost/ || exit 1"},
			expectFlags: map[string]string{"start-period": "5s", "interval": "5s"},
		},
		// 4. no interval
		{
			line: &parser.Line{
				Command: "HEALTHCHECK",
				Cells: []*parser.Cell{
					{
						Value: "CMD",
					},
					{
						Value: "curl -fs http://localhost/ || exit 1",
					},
				},
				Flags: map[string]string{"start-period": "5s"},
			},
			expectArgs:  []string{"CMD", "curl -fs http://localhost/ || exit 1"},
			expectFlags: map[string]string{"start-period": "5s"},
		},
		// 5. all default
		{
			line: &parser.Line{
				Command: "HEALTHCHECK",
				Cells: []*parser.Cell{
					{
						Value: "CMD",
					},
					{
						Value: "curl -fs http://localhost/ || exit 1",
					},
				},
				Flags: map[string]string{},
			},
			expectArgs:  []string{"CMD", "curl -fs http://localhost/ || exit 1"},
			expectFlags: map[string]string{},
		},
	}

	for _, tc := range testcases {
		allowFlags := map[string]bool{"start-period": true, "interval": true, "timeout": true, "retries": true}
		flags, args := getFlagsAndArgs(tc.line, allowFlags)
		assert.DeepEqual(t, flags, tc.expectFlags)
		assert.DeepEqual(t, args, tc.expectArgs)
	}
}

// FROM alpine:latest
// FROM docker.io/alpine:2.0
// FROM alpine@digest
// FROM alpine:latest@digest <- fail
// FROM alpine:latest:latest <- fail
// FROM alpine@digest@digest <- fail
func TestResolveImageName(t *testing.T) {
	type args struct {
		s   string
		reg *regexp.Regexp
	}
	tests := []struct {
		name    string
		args    args
		ret     string
		wantErr bool
	}{
		{
			name:    "test 1",
			args:    args{s: "alpine:latest"},
			ret:     "alpine:latest",
			wantErr: false,
		},
		{
			name:    "test 2",
			args:    args{s: "alpine:3.2"},
			ret:     "alpine:3.2",
			wantErr: false,
		},
		{
			name:    "test 3",
			args:    args{s: "alpine@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72"},
			ret:     "alpine@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72",
			wantErr: false,
		},
		{
			name:    "test 4 - with tag and digest",
			args:    args{s: "alpine:3.2@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72"},
			ret:     "alpine:3.2@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72",
			wantErr: false,
		},
		{
			name:    "test 6 - incorrect format with 2 * digests",
			args:    args{s: "alpine@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72"},
			ret:     "",
			wantErr: true,
		},
		{
			name:    "test 7",
			args:    args{s: "docker.io/busybox:monica"},
			ret:     "docker.io/busybox:monica",
			wantErr: false,
		},
		{
			name:    "test 8",
			args:    args{s: "localhost:8080/busybox:monica"},
			ret:     "localhost:8080/busybox:monica",
			wantErr: false,
		},
		{
			name:    "test 9",
			args:    args{s: "127.0.0.1:8080/busybox:monica"},
			ret:     "127.0.0.1:8080/busybox:monica",
			wantErr: false,
		},
		{
			name:    "test 10",
			args:    args{s: "127.0.0.1:8080/alpine@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72"},
			ret:     "127.0.0.1:8080/alpine@sha256:a187dde48cd289ac374ad8539930628314bc581a481cdb41409c9289419ddb72",
			wantErr: false,
		},
		{
			name:    "test 11",
			args:    args{s: "non-existent-registry.com/alpine"},
			ret:     "non-existent-registry.com/alpine",
			wantErr: false,
		},
		{
			name:    "test 12",
			args:    args{s: "golang:1.11.1-alpine3.8"},
			ret:     "golang:1.11.1-alpine3.8",
			wantErr: false,
		},
		{
			name:    "test 13",
			args:    args{s: "registry.centos.org/centos/centos:centos7"},
			ret:     "registry.centos.org/centos/centos:centos7",
			wantErr: false,
		},
		{
			name:    "test 14",
			args:    args{s: "testuser/busybox:git-base@sha256:b39333801e01efb8cb3e346aa866879107dffacdb83e555f877d6e7215db14da"},
			ret:     "testuser/busybox:git-base@sha256:b39333801e01efb8cb3e346aa866879107dffacdb83e555f877d6e7215db14da",
			wantErr: false,
		},
		{
			name:    "test 15",
			args:    args{s: "registry:2"},
			ret:     "registry:2",
			wantErr: false,
		},
		{
			name:    "test 21",
			args:    args{s: "https://www.dropbox.com/s/kpbrx26bwhoa1rp/moment.js?raw=1"},
			ret:     "",
			wantErr: true,
		},
		{
			name:    "test 22",
			args:    args{s: "https://hub.docker.com/_/busybox"},
			ret:     "",
			wantErr: true,
		},
		{
			name:    "test 23",
			args:    args{s: "hub.docker.com/_/busybox"},
			ret:     "",
			wantErr: true,
		},
		{
			name:    "test 24",
			args:    args{s: "https://"},
			ret:     "",
			wantErr: true,
		},
		{
			name:    "test 25",
			args:    args{s: "Busybox"},
			ret:     "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				buildOpts: BuildOptions{
					BuildArgs: map[string]string{},
				},
				cliLog:      logger.NewCliLogger(constant.CliLogBufferLen),
				headingArgs: map[string]string{},
				unusedArgs:  map[string]string{},
			}
			ret, err := image.ResolveImageName(tt.args.s, b.searchArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveImageName() error = %v, wantErr %v", err, tt.wantErr)
				t.FailNow()
			}
			if tt.ret != ret {
				t.Errorf("Failed: expected ret: %q, got: %q", tt.ret, ret)
				t.FailNow()
			}
		})
	}
}

// FROM $imageName
// FROM ${imageName}
// FROM {imageName}  <- fail
// FROM ${imageName}:$tag
// FROM $imageName@${digest}
// FROM ${imageName}${namePart2}$namePart3$namePart4
// FROM $imageName@${digest}$ <- fail
// FROM $imageName@${digest}$${digest2} <- fail
func TestResolveImageNameWithArgs(t *testing.T) {
	type args struct {
		s   string
		reg *regexp.Regexp
	}
	tests := []struct {
		name        string
		args        args
		ret         string
		wantErr     bool
		headingArgs map[string]string
		unusedArgs  map[string]string
		funcCheck   func(t *testing.T, b *Builder)
	}{
		{
			name:        "test 1",
			args:        args{s: "${imageName}"},
			ret:         "alpine",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{"imageName": "alpine", "imageName2": "busybox"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"imageName2": "busybox"})
			},
		},
		{
			name:        "test 2",
			args:        args{s: "$imageName"},
			ret:         "alpine",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "alpine", "imageName2": "busybox"},
			unusedArgs:  map[string]string{"imageName": "alpine", "imageName2": "busybox"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"imageName2": "busybox"})
			},
		},
		{
			name:        "test 3",
			args:        args{s: "$imageName:$tag"},
			ret:         "euleros:2.0",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "euleros", "tag": "2.0"},
			unusedArgs:  map[string]string{"imageName2": "busybox"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"imageName2": "busybox"})
			},
		},
		{
			name:        "test 4",
			args:        args{s: "$imageName@${digest}"},
			ret:         "openeuler@sha256:bc3e77282fe1d51402141daad38fed0226c15f6de18243faf64de4f5d67347e1",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "openeuler", "tag": "20.03", "digest": "sha256:bc3e77282fe1d51402141daad38fed0226c15f6de18243faf64de4f5d67347e1"},
			unusedArgs: map[string]string{"imageName": "openeuler", "tag": "20.03",
				"digest": "bc3e77282fe1d51402141daad38fed0226c15f6de18243faf64de4f5d67347e1"},
			funcCheck: func(t *testing.T, b *Builder) {
				assert.DeepEqual(t, b.unusedArgs, map[string]string{"tag": "20.03"})
			},
		},
		{
			name:        "test 5",
			args:        args{s: "${imageName}${namePart2}$namePart3$namePart4"},
			ret:         "n1n2p3p4",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "n1", "namePart2": "n2", "namePart3": "p3", "namePart4": "p4"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 6",
			args:        args{s: "${imageName}-${namePart2}_$namePart3/$namePart4"},
			ret:         "n1-n2_p3/p4",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "n1", "namePart2": "n2", "namePart3": "p3", "namePart4": "p4"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 7",
			args:        args{s: "$imageName@${digest}$"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 8",
			args:        args{s: "$imageName@${digest}$${digest2}"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 9",
			args:        args{s: "${imageName"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 10",
			args:        args{s: "${imageName$namePart2"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			// FROM doesn't accept "\$"
			name:        "test 11",
			args:        args{s: "\\${imageName}"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 12 - normal with :-",
			args:        args{s: "${imageName:-busybox}"},
			ret:         "busybox",
			wantErr:     false,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 13 - normal with :-",
			args:        args{s: "${imageName:-busybox}"},
			ret:         "alpine",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 14 - normal with :+",
			args:        args{s: "${imageName:+busybox}"},
			ret:         "",
			wantErr:     true, // args will be resolved to "", which is not allowed
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 15 - normal with :+",
			args:        args{s: "${imageName:+busybox}"},
			ret:         "busybox",
			wantErr:     false,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 16 - abnormal with :=",
			args:        args{s: "${imageName:=busybox}"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 17 - abnormal with :-",
			args:        args{s: "$imageName:-busybox"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{"imageName": "alpine"},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 18 - abnormal with :+",
			args:        args{s: "$imageName:+busybox"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 19 - abnormal with incorrect arg",
			args:        args{s: "$1imageName"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
		{
			name:        "test 20 - abnormal with incorrect arg",
			args:        args{s: "$-imageName"},
			ret:         "",
			wantErr:     true,
			headingArgs: map[string]string{},
			unusedArgs:  map[string]string{},
			funcCheck:   func(t *testing.T, b *Builder) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				cliLog:      logger.NewCliLogger(constant.CliLogBufferLen),
				headingArgs: tt.headingArgs,
				unusedArgs:  tt.unusedArgs,
			}
			ret, err := image.ResolveImageName(tt.args.s, b.searchArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveImageName() error = %v, wantErr %v", err, tt.wantErr)
				t.FailNow()
			}
			if tt.ret != ret {
				t.Errorf("Failed: expected ret: %q, got: %q", tt.ret, ret)
				t.FailNow()
			}
			tt.funcCheck(t, b)
		})
	}
}

func TestWriteImageId(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	imageIDFilePath := tmpDir.Join("iidfile")
	b := &Builder{
		buildOpts: BuildOptions{
			Iidfile: imageIDFilePath,
		},
		cliLog: logger.NewCliLogger(constant.CliLogBufferLen),
	}

	imageID := "38b993607bcabe01df1dffdf01b329005c6a10a36d557f9d073fc25943840c66"

	err := b.writeImageID(imageID)
	assert.NilError(t, err)
	bytes, err := ioutil.ReadFile(imageIDFilePath)
	assert.NilError(t, err)
	assert.Equal(t, imageID, string(bytes))
}

func TestSearchArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		headArgs map[string]string
		ret      string
	}{
		{
			name:     "test 1",
			arg:      "alpine:latest",
			headArgs: map[string]string{},
			ret:      "",
		},
		{
			name:     "test 2",
			arg:      "testArg:-v1.0",
			headArgs: map[string]string{"testArg": "vvv"},
			ret:      "vvv",
		},
		{
			name:     "test 3",
			arg:      "testArg:-v1.0",
			headArgs: map[string]string{},
			ret:      "v1.0",
		},
		{
			name:     "test 4",
			arg:      "testArg:-",
			headArgs: map[string]string{},
			ret:      "",
		},
		{
			name:     "test 10",
			arg:      "testArg:+v1.0",
			headArgs: map[string]string{"testArg": "vvv"},
			ret:      "v1.0",
		},
		{
			name:     "test 11",
			arg:      "testArg:+v1.0",
			headArgs: map[string]string{},
			ret:      "",
		},
		{
			name:     "test 12",
			arg:      "testArg:+",
			headArgs: map[string]string{},
			ret:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				headingArgs: tt.headArgs,
			}
			assert.Equal(t, tt.ret, b.searchArg(tt.arg))
		})
	}
}

func TestParseRequestBuildArgs(t *testing.T) {
	var tests = []struct {
		name      string
		buildArgs []string
		decrypt   bool
		wantErr   bool
	}{
		{
			name:      "case 1 - no build-args",
			buildArgs: []string{},
			decrypt:   false,
			wantErr:   false,
		},
		{
			name:      "case 2 - normal build-args",
			buildArgs: []string{"foo=bar"},
			decrypt:   false,
			wantErr:   false,
		},
		{
			name:      "case 3 - build-args needs decrypt",
			buildArgs: []string{"foo=bar", "http_proxy=test"},
			decrypt:   true,
			wantErr:   false,
		},
	}

	b := getBuilder()
	rsaKey := b.rsaKey
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				argsParsed map[string]string
				err        error
			)
			if tt.decrypt {
				b.rsaKey = rsaKey
				tmpDir := fs.NewDir(t, t.Name())
				defer tmpDir.Remove()
				keyPath := filepath.Join(tmpDir.Path(), "isula-build.pub")
				assert.NilError(t, err)
				err = util.GenRSAPublicKeyFile(b.rsaKey, keyPath)
				assert.NilError(t, err)
				pubKey, err := util.ReadPublicKey(keyPath)
				assert.NilError(t, err)
				var args = make([]string, 0, 10)
				for _, v := range tt.buildArgs {
					encryptedArg, encErr := util.EncryptRSA(v, pubKey, sha512.New())
					assert.NilError(t, encErr)
					args = append(args, encryptedArg)
				}
				argsParsed, err = b.parseBuildArgs(args, tt.decrypt)
				b.buildOpts.BuildArgs = argsParsed
			} else {
				b.rsaKey = &rsa.PrivateKey{}
				argsParsed, err = b.parseBuildArgs(tt.buildArgs, tt.decrypt)
				b.buildOpts.BuildArgs = argsParsed
			}

			if (err == nil) != (!tt.wantErr) {
				t.FailNow()
			}

			var argsMap = make(map[string]string, len(tt.buildArgs))
			for _, arg := range tt.buildArgs {
				av := strings.SplitN(arg, "=", 2)
				argsMap[av[0]] = av[1]
			}
			assert.DeepEqual(t, argsMap, b.buildOpts.BuildArgs)

			b.buildOpts.BuildArgs = nil
		})
	}
}

func TestParseTag(t *testing.T) {
	type testcase struct {
		name   string
		output string
		tag    string
	}
	testcases := []testcase{
		{
			name:   "docker-daemon output",
			output: "docker-daemon:isula/test:latest",
			tag:    "isula/test:latest",
		},
		{
			name:   "isulad output",
			output: "isulad:isula/test:latest",
			tag:    "isula/test:latest",
		},
		{
			name:   "docker-archive output",
			output: "docker-archive:./isula.tar:isula:latest",
			tag:    "isula:latest",
		},
		{
			name:   "docker-archive output without tag",
			output: "docker-archive:./isula.tar",
			tag:    "",
		},
		{
			name:   "docker-archive output with long tag",
			output: "docker-archive:./isula.tar:aaa:bbb:ccc",
			tag:    "aaa:bbb:ccc",
		},
		{
			name:   "docker-archive output only with name",
			output: "docker-archive:./isula.tar:isula",
			tag:    "isula",
		},
		{
			name:   "docker output",
			output: "docker://localhost:5000/isula/test:latest",
			tag:    "isula/test:latest",
		},
		{
			name:   "docker output",
			output: "docker://localhost:5000/isula/test",
			tag:    "isula/test",
		},
		{
			name:   "invalid docker output",
			output: "docker:localhost:5000/isula/test:latest",
			tag:    "",
		},
	}
	for _, tc := range testcases {
		tag := parseOutputTag(tc.output)
		assert.Equal(t, tag, tc.tag, tc.name)
	}
}

func TestNewBuilder(t *testing.T) {
	// tmpfs doesn't not support chattr +i to immutable
	tmpDir, err := ioutil.TempDir("/var/tmp", t.Name())
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer os.RemoveAll(tmpDir)
	immutablePath := filepath.Join(tmpDir, "run")
	os.Mkdir(immutablePath, 0644)
	if err = testutil.Immutable(immutablePath, true); err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer testutil.Immutable(immutablePath, false)

	keyPath := filepath.Join(tmpDir, "isula-build.pub")
	privateKey, err := util.GenerateRSAKey(util.DefaultRSAKeySize)
	assert.NilError(t, err)
	err = util.GenRSAPublicKeyFile(privateKey, keyPath)
	assert.NilError(t, err)

	localStore, err := store.GetStore()
	assert.NilError(t, err)

	type args struct {
		ctx         context.Context
		store       store.Store
		req         *pb.BuildRequest
		runtimePath string
		buildDir    string
		runDir      string
		key         *rsa.PrivateKey
	}
	tests := []struct {
		name    string
		args    args
		want    *Builder
		wantErr bool
	}{
		{
			name: "NewBuilder - wrong rundir",
			args: args{
				ctx:      context.Background(),
				store:    localStore,
				req:      &pb.BuildRequest{},
				buildDir: tmpDir,
				runDir:   "",
				key:      privateKey,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NewBuilder - parseOutput fail",
			args: args{
				ctx:   context.Background(),
				store: localStore,
				req: &pb.BuildRequest{
					Output: "docker-archive:/home/test/aa.tar",
				},
				buildDir: tmpDir,
				runDir:   immutablePath,
				key:      privateKey,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBuilder(tt.args.ctx, &tt.args.store, tt.args.req, tt.args.runtimePath, tt.args.buildDir, tt.args.runDir, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBuilder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBuilder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAndExpandTag(t *testing.T) {
	type testcase struct {
		name    string
		tag     string
		output  string
		wantErr bool
	}
	testcases := []testcase{
		{
			name:    "test 1",
			tag:     "isula/test",
			output:  "isula/test:latest",
			wantErr: false,
		},
		{
			name:    "test 2",
			tag:     "localhost:5000/test",
			output:  "localhost:5000/test:latest",
			wantErr: false,
		},
		{
			name:    "test 3",
			tag:     "isula/test:latest",
			output:  "isula/test:latest",
			wantErr: false,
		},
		{
			name:    "test 4",
			tag:     "localhost:5000/test:latest",
			output:  "localhost:5000/test:latest",
			wantErr: false,
		},
		{
			name:    "test 5",
			tag:     "localhost:5000:aaa/test:latest",
			output:  "",
			wantErr: true,
		},
		{
			name:    "test 6",
			tag:     "localhost:5000:aaa/test",
			output:  "",
			wantErr: true,
		},
		{
			name:    "test 7",
			tag:     "localhost:5000/test:latest:latest",
			output:  "",
			wantErr: true,
		},
		{
			name:    "test 8",
			tag:     "test:latest:latest",
			output:  "",
			wantErr: true,
		},
		{
			name:    "test 9",
			tag:     "",
			output:  "<none>:<none>",
			wantErr: false,
		},
		{
			name:    "test 10",
			tag:     "abc efg:latest",
			output:  "",
			wantErr: true,
		},
		{
			name:    "test 10",
			tag:     "abc!@#:latest",
			output:  "",
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		_, tag, err := CheckAndExpandTag(tc.tag)
		assert.Equal(t, tag, tc.output, tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("getCheckAndExpandTag() error = %v, wantErr %v", err, tc.wantErr)
		}
	}
}
