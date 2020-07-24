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
// Description: cmdBuilder related functions tests

package dockerfile

import (
	"context"
	"testing"
	"time"

	"github.com/containers/image/v5/pkg/strslice"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
	"github.com/docker/docker/pkg/signal"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/util"
)

func init() {
	reexec.Init()
}

func TestExecuteHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		fromConfig  *docker.Image
		attribute   string
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name:        "cmd normal test with normal cmd",
			fileContent: "FROM alpine\nHEALTHCHECK --interval=5m --timeout=3s CMD curl -f http://localhost/ || exit 1",
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Healthcheck: &docker.HealthConfig{}}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				expect := &docker.HealthConfig{
					Test:        []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
					Interval:    time.Duration(5000000000 * 60),
					StartPeriod: time.Duration(0),
					Timeout:     time.Duration(3000000000),
					Retries:     3,
				}
				assert.DeepEqual(t, s.docker.Config.Healthcheck, expect)
			},
		},
		{
			name:        "cmd normal test with json",
			fileContent: "FROM alpine\nHEALTHCHECK --interval=5m --timeout=3s CMD [\"pwd\"]",
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Healthcheck: &docker.HealthConfig{}}}},
			attribute:   "json",
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				expect := &docker.HealthConfig{
					Test:        []string{"CMD", "pwd"},
					Interval:    time.Duration(5000000000 * 60),
					StartPeriod: time.Duration(0),
					Timeout:     time.Duration(3000000000),
					Retries:     3,
				}
				assert.DeepEqual(t, s.docker.Config.Healthcheck, expect)
			},
		},
		{
			name:        "none mode",
			fileContent: "FROM alpine\nHEALTHCHECK none",
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Healthcheck: &docker.HealthConfig{}}}},
			attribute:   "json",
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				expect := &docker.HealthConfig{
					Test:        []string{"NONE"},
					Interval:    time.Duration(0),
					StartPeriod: time.Duration(0),
					Timeout:     time.Duration(0),
					Retries:     0,
				}
				assert.DeepEqual(t, s.docker.Config.Healthcheck, expect)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:       make(map[string]string),
				rawStage:  generateOneRawStage(t, tt.fileContent),
				docker:    tt.fromConfig,
				shellForm: strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			if err := s.commands[0].cmdExecutor(); (err != nil) != tt.wantErr {
				t.Errorf("CmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteCmd(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		fromConfig  *docker.Image
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name:        "normal test - type shell",
			fileContent: "FROM alpine\nCMD /bin/sh -c sleep 1",
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"/bin/sh", "-c", "/bin/sh -c sleep 1"})
			},
		},
		{
			name: "normal test - type exec 1",
			fileContent: `FROM alpine
CMD ["/bin/sh", "-c", "sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"/bin/sh", "-c", "sleep", "1"})
			},
		},
		{
			name: "normal test - type exec 2",
			fileContent: `FROM alpine
CMD ["sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"sleep", "1"})
			},
		},
		{
			// strange input but works at Docker
			name: "normal test but strange - type shell",
			fileContent: `FROM alpine
CMD ["/bin/sh", -c, "sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"/bin/sh", "-c", "[\"/bin/sh\", -c, \"sleep\", \"1\"]"})
			},
		},
		{
			name: "normal test - empty input 1",
			fileContent: `FROM alpine
CMD [""]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:       make(map[string]string),
				rawStage:  generateOneRawStage(t, tt.fileContent),
				docker:    tt.fromConfig,
				shellForm: strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			if err := s.commands[0].cmdExecutor(); (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestNewCmdBuilder(t *testing.T) {
	line := &parser.Line{
		Command: "ENV",
		Raw:     "http_proxy=1.1.1.1",
	}
	cb := newCmdBuilder(context.Background(), line, &stageBuilder{}, nil, nil)
	assert.Assert(t, cb != nil)
	assert.Equal(t, cb.line.Command, line.Command) // nolint:staticcheck
}

func TestCmdBuilderCommit(t *testing.T) {
	line := &parser.Line{
		Command: "ENV",
		Raw:     "s=1111",
	}

	ctx := context.WithValue(context.Background(), util.LogFieldKey(util.LogKeyBuildID), "0123456789")
	ctx = context.WithValue(ctx, util.BuildDirKey(util.BuildDir), "/tmp/isula-build-test")
	s := &stageBuilder{
		localStore: localStore,
		builder: &Builder{
			cliLog: logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:    ctx,
		},
	}

	cb := newCmdBuilder(context.Background(), line, &stageBuilder{
		localStore: localStore,
		builder:    &Builder{cliLog: logger.NewCliLogger(constant.CliLogBufferLen), ctx: ctx},
	}, nil, nil)
	assert.Assert(t, cb != nil)

	tmpName := stringid.GenerateRandomID() + "test"
	container, err := cb.stage.localStore.CreateContainer("", []string{tmpName}, "", "", "", nil)
	defer func() {
		err = cb.stage.localStore.DeleteContainer(container.ID)
		assert.NilError(t, err)
		s.builder.cleanup()
	}()
	assert.NilError(t, err)
	assert.Assert(t, container != nil)
	cb.stage.containerID = container.ID
	cb.stage.docker = &docker.Image{}
	image.UpdateV2Image(cb.stage.docker)

	imgID, err := cb.commit(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(imgID) > 0)
}

func TestExecuteShell(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		fromConfig  *docker.Image
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "test 1",
			fileContent: `FROM alpine
SHELL ["/bin/bash", "-c"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.shellForm, strslice.StrSlice{"/bin/bash", "-c"})
				assert.DeepEqual(t, s.docker.Config.Shell, strslice.StrSlice{"/bin/bash", "-c"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:       make(map[string]string),
				rawStage:  generateOneRawStage(t, tt.fileContent),
				docker:    tt.fromConfig,
				shellForm: strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			if err := s.commands[0].cmdExecutor(); (err != nil) != tt.wantErr {
				t.Errorf("SHELL cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteShellAndCmd(t *testing.T) {
	fileContent := `FROM alpine
CMD ls
SHELL ["/bin/bash", "-c"]
CMD ls`
	fromConfig := &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}}
	s := &stageBuilder{
		builder: &Builder{
			reservedArgs: make(map[string]string),
			cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:          context.Background(),
		},
		env:       make(map[string]string),
		rawStage:  generateOneRawStage(t, fileContent),
		docker:    fromConfig,
		shellForm: strslice.StrSlice{"/bin/sh", "-c"},
	}
	err := s.analyzeStage(context.Background())
	assert.NilError(t, err)

	if err := s.commands[0].cmdExecutor(); err != nil {
		t.Errorf("CMD cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"/bin/sh", "-c", "ls"})

	if err := s.commands[1].cmdExecutor(); err != nil {
		t.Errorf("SHELL cmdExecutor() error: %v", err)
	}
	if err := s.commands[2].cmdExecutor(); err != nil {
		t.Errorf("CMD cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.shellForm, strslice.StrSlice{"/bin/bash", "-c"})
	assert.DeepEqual(t, s.docker.Config.Shell, strslice.StrSlice{"/bin/bash", "-c"})
	assert.DeepEqual(t, s.docker.Config.Cmd, strslice.StrSlice{"/bin/bash", "-c", "ls"})
}

func TestExecuteNoop(t *testing.T) {
	s := &stageBuilder{
		builder: &Builder{
			reservedArgs: make(map[string]string),
			cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:          context.Background(),
		},
		env:      make(map[string]string),
		rawStage: generateOneRawStage(t, "FROM alpine\nARG testArg\nENV env1=env2\nONBUILD CMD ls"),
		docker:   &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
	}
	err := s.analyzeStage(context.Background())
	assert.NilError(t, err)
	for _, cmd := range s.commands {
		if err := cmd.cmdExecutor(); err != nil {
			t.Errorf("CmdExecutor() error: %v", err)
		}
	}

	s.builder.cleanup()

	var stepPrints string
	for s := range s.builder.StatusChan() {
		stepPrints += s
	}

	// the "STEP 1: FROM alpine" in production is done at stageBuilder.prepare()
	// no cmdExecutor for FROM, so no print for FROM here
	expectedString := `STEP  1: ARG testArg
STEP  2: ENV env1=env2
STEP  3: ONBUILD CMD ls
`
	assert.Equal(t, stepPrints, expectedString)
}

func TestExecuteEntrypoint(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		fromConfig  *docker.Image
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name:        "normal test - type shell",
			fileContent: "FROM alpine\nENTRYPOINT /bin/sh -c sleep 1",
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Entrypoint, strslice.StrSlice{"/bin/sh", "-c", "/bin/sh -c sleep 1"})
			},
		},
		{
			name: "normal test - type exec",
			fileContent: `FROM alpine
ENTRYPOINT ["/bin/sh", "-c", "sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Entrypoint, strslice.StrSlice{"/bin/sh", "-c", "sleep", "1"})
			},
		},
		{
			name: "normal test - type exec 2",
			fileContent: `FROM alpine
ENTRYPOINT ["sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Entrypoint, strslice.StrSlice{"sleep", "1"})
			},
		},
		{
			// strange input but works at Docker
			name: "normal test but strange - type shell",
			fileContent: `FROM alpine
ENTRYPOINT ["/bin/sh", -c, "sleep", "1"]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Entrypoint, strslice.StrSlice{"/bin/sh", "-c", "[\"/bin/sh\", -c, \"sleep\", \"1\"]"})
			},
		},
		{
			name: "normal test - empty input 1",
			fileContent: `FROM alpine
ENTRYPOINT [""]`,
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Entrypoint, strslice.StrSlice{})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:       make(map[string]string),
				rawStage:  generateOneRawStage(t, tt.fileContent),
				docker:    tt.fromConfig,
				shellForm: strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			if err := s.commands[0].cmdExecutor(); (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteVolume(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		buildArgs   map[string]string
		fromConfig  *docker.Image
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name:        "normal test",
			fileContent: "FROM alpine\nVOLUME /log",
			buildArgs:   make(map[string]string),
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"/log": {}})
			},
		},
		{
			name:        "normal test 2",
			fileContent: "FROM alpine\nVOLUME /logbk",
			buildArgs:   make(map[string]string),
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{"/log": {}}}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"/log": {}, "/logbk": {}})
			},
		},
		{
			name:        "normal test 3",
			fileContent: "FROM alpine\nVOLUME vol1 vol2",
			buildArgs:   make(map[string]string),
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:     false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"vol1": {}, "vol2": {}})
			},
		},
		{
			name: "normal test 4",
			fileContent: `FROM alpine
VOLUME ["/data1","/data2"]`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"/data1": {}, "/data2": {}})
			},
		},
		{
			name:        "normal test 5",
			fileContent: "FROM alpine\nVOLUME $vol1 ${vol2}",
			buildArgs:   map[string]string{"vol1": "/usr/test1", "vol2": "/tmp/test2"},
			fromConfig:  &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:     true,
			funcCheck:   func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "normal test 6",
			fileContent: `FROM alpine
ARG vol1
ARG vol2
VOLUME $vol1 ${vol2}`,
			buildArgs:  map[string]string{"vol1": "/usr/test1", "vol2": "/tmp/test2"},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"/usr/test1": {}, "/tmp/test2": {}})
			},
		},
		{
			name: "normal test 7",
			fileContent: `FROM alpine
ARG vol1
VOLUME ["/$vol1","${vol2}/test"]`,
			buildArgs:  map[string]string{"vol1": "test1", "vol2": "/tmp"},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Volumes: map[string]struct{}{}}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Volumes, map[string]struct{}{"/test1": {}, "/test": {}})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := fs.NewDir(t, "TestExecuteVolume")
			defer dir.Remove()
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				mountpoint: dir.Path(),
				env:        make(map[string]string),
				rawStage:   generateOneRawStage(t, tt.fileContent),
				docker:     tt.fromConfig,
				shellForm:  strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			var retErr error
			for _, cmd := range s.commands {
				if retErr = cmd.cmdExecutor(); retErr != nil {
					break
				}
			}
			if (retErr != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", retErr, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteLabel(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		buildArgs   map[string]string
		fromConfig  *docker.Image
		wantErr     bool
		funcCheck   func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "normal test 1",
			fileContent: `FROM alpine
LABEL "com.example.vendor"="Foo Incorporated"`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels, map[string]string{"com.example.vendor": "Foo Incorporated"})
			},
		},
		{
			name: "normal test 2",
			fileContent: `FROM alpine
LABEL com.example.label-with-value="foo"`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels, map[string]string{"com.example.label-with-value": "foo"})
			},
		},
		{
			name: "normal test 3",
			fileContent: `FROM alpine
LABEL version="1.0"`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels, map[string]string{"version": "1.0"})
			},
		},
		{
			name: "normal test 4",
			fileContent: `FROM alpine
LABEL multi.label1="value1" multi.label2="value2" other="value3"`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels,
					map[string]string{"multi.label1": "value1", "multi.label2": "value2", "other": "value3"})
			},
		},
		{
			name: "normal test 5",
			fileContent: `FROM alpine
LABEL multi.label1="$arg1" multi.label2="${arg2}" other="$arg3"`,
			buildArgs:  make(map[string]string),
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels,
					map[string]string{"multi.label1": "", "multi.label2": "", "other": ""})
			},
		},
		{
			name: "normal test 6",
			fileContent: `FROM alpine
ARG arg1
ARG arg2=value2
LABEL multi.label1="$arg1" multi.label2="${arg2}" other="$arg3"`,
			buildArgs:  map[string]string{"arg1": "value1"},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Labels,
					map[string]string{"multi.label1": "value1", "multi.label2": "value2", "other": ""})
			},
		},
		{
			name: "abnormal test 10",
			fileContent: `FROM alpine
LABEL $=0`,
			buildArgs:  map[string]string{},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    true,
			funcCheck:  func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "abnormal test 11",
			fileContent: `FROM alpine
LABEL $=$`,
			buildArgs:  map[string]string{},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    true,
			funcCheck:  func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "abnormal test 12",
			fileContent: `FROM alpine
LABEL $=#% $#v=* #*=@#$%`,
			buildArgs:  map[string]string{},
			fromConfig: &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{Labels: make(map[string]string)}}},
			wantErr:    true,
			funcCheck:  func(t *testing.T, s *stageBuilder) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:       make(map[string]string),
				rawStage:  generateOneRawStage(t, tt.fileContent),
				docker:    tt.fromConfig,
				shellForm: strslice.StrSlice{"/bin/sh", "-c"},
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			var retErr error
			for _, cmd := range s.commands {
				if retErr = cmd.cmdExecutor(); retErr != nil {
					break
				}
			}
			if (retErr != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", retErr, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

// docker will accept any string wrote in WORKDIR filed
func TestExecuteWorkDir(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "WORKDIR handler test 1 - one workdir",
			dockerfile: `FROM alpine AS cho
			WORKDIR /path/to/your/directory`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/path/to/your/directory")
			},
		},
		{
			// strange but work for docker
			name: "WORKDIR handler test 2 - strange workdir",
			dockerfile: `FROM alpine AS cho
				WORKDIR !@#!@F!#$T!%!@$# " "`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/!@#!@F!#!%!@  ")
			},
		},
		{
			// strange but work for docker
			name: "WORKDIR handler test 3 - strange relative workdir",
			dockerfile: `FROM alpine AS cho
				WORKDIR ../../../../../`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/")
			},
		},
		{
			name: "WORKDIR handler test 4 - with not defined param",
			dockerfile: `FROM alpine AS cho
				WORKDIR $DIRPATH/$DIRNAME`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/")
			},
		},
		{
			name: "WORKDIR handler test 5 - with defined param",
			dockerfile: `FROM alpine AS cho
				ARG DIRPATH
				ARG DIRNAME
				WORKDIR $DIRPATH/$DIRNAME`,
			buildArgs: map[string]string{"DIRPATH": "/var", "DIRNAME": "run"},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/var/run")
			},
		},
		{
			name:       "WORKDIR handler test 6 - special ",
			dockerfile: "FROM alpine\nWORKDIR % ` *db a`a c )",
			buildArgs:  map[string]string{},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/% ` *db a`a c )")
			},
		},
		{
			name: "WORKDIR handler test 7 - special ",
			dockerfile: `FROM alpine
				WORKDIR ' ' 'db a'`,
			buildArgs: map[string]string{},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/  db a")
			},
		},
		{
			name: "WORKDIR handler test 8 - special ",
			dockerfile: `FROM alpine
				WORKDIR " " "db a"`,
			buildArgs: map[string]string{},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.WorkingDir, "/  db a")
			},
		},
		{
			name:       "WORKDIR handler test 9 - special ",
			dockerfile: "FROM alpine\nWORKDIR % \" *db a`a c )",
			buildArgs:  map[string]string{},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr:   true,
			funcCheck: func(t *testing.T, s *stageBuilder) {},
		},
	}
	logrus.SetLevel(logrus.DebugLevel)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := fs.NewDir(t, "TestExecuteWorkDir")
			defer dir.Remove()
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				mountpoint: dir.Path(),
				rawStage:   generateOneRawStage(t, tt.dockerfile),
				docker:     tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			var retErr error
			for _, cmd := range s.commands {
				if retErr = cmd.cmdExecutor(); retErr != nil {
					break
				}
			}
			if (retErr != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", retErr, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestMultipleAbsWorkDir(t *testing.T) {
	dockerfile := `FROM alpine AS cho
WORKDIR /a
WORKDIR /b
WORKDIR /c`
	config := &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}}
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	s := &stageBuilder{
		builder: &Builder{
			reservedArgs: make(map[string]string),
			cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:          context.Background(),
		},
		mountpoint: dir.Path(),
		env:        make(map[string]string),
		rawStage:   generateOneRawStage(t, dockerfile),
		docker:     config,
	}
	err := s.analyzeStage(context.Background())
	assert.NilError(t, err)
	if err := s.commands[0].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/a")
	if err := s.commands[1].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/b")
	if err := s.commands[2].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/c")
}

func TestMultipleRelativeWorkDir(t *testing.T) {
	dockerfile := `FROM alpine AS cho
WORKDIR /a
WORKDIR b
WORKDIR c`
	config := &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}}
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	s := &stageBuilder{
		builder: &Builder{
			reservedArgs: make(map[string]string),
			cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:          context.Background(),
		},
		mountpoint: dir.Path(),
		env:        make(map[string]string),
		rawStage:   generateOneRawStage(t, dockerfile),
		docker:     config,
	}
	err := s.analyzeStage(context.Background())
	assert.NilError(t, err)
	if err := s.commands[0].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/a")
	if err := s.commands[1].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/a/b")
	if err := s.commands[2].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/a/b/c")
}

func TestWoriDirWithVariable(t *testing.T) {
	dockerfile := `FROM alpine AS cho
ENV DIRPATH /path
ARG DIRNAME=mypath
WORKDIR $DIRPATH/$DIRNAME`
	config := &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}}
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	s := &stageBuilder{
		builder: &Builder{
			reservedArgs: make(map[string]string),
			cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
			ctx:          context.Background(),
		},
		mountpoint: dir.Path(),
		env:        make(map[string]string),
		rawStage:   generateOneRawStage(t, dockerfile),
		docker:     config,
	}
	err := s.analyzeStage(context.Background())
	assert.NilError(t, err)
	if err = s.commands[0].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.NilError(t, err)
	if err = s.commands[1].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.NilError(t, err)
	if err = s.commands[2].cmdExecutor(); err != nil {
		t.Errorf("WORKDIR cmdExecutor() error: %v", err)
	}
	assert.NilError(t, err)
	assert.DeepEqual(t, s.docker.Config.WorkingDir, "/path/mypath")
}

// docker will accept any string wrote in MANTAINER filed
func TestExecuteMaintainer(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "Maintainer handler test 1",
			dockerfile: `FROM alpine
Maintainer iSula iSula@huawei.com`,
			config: &docker.Image{
				V1Image: docker.V1Image{
					Author: "",
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Author, "iSula iSula@huawei.com")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			if err := s.commands[0].cmdExecutor(); (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestValidateSignal(t *testing.T) {
	tests := []struct {
		name    string
		sig     string
		wantErr bool
	}{
		{
			name: "validateSignal test 1 - normal string sig",
			sig:  "kill",
		},
		{
			name: "validateSignal test 2 - normal int sig",
			sig:  "9",
		},
		{
			name:    "validateSignal test 3 - abnormal big int sig",
			sig:     "65",
			wantErr: true,
		},
		{
			name:    "validateSignal test 4 - abnormal string sig",
			sig:     "SIG",
			wantErr: true,
		},
		{
			name:    "validateSignal test 5 - abnormal int sig",
			sig:     "0",
			wantErr: true,
		},
		{
			name:    "validateSignal test 6 - abnormal empty sig",
			sig:     "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := util.ValidateSignal(tt.sig); (err != nil) != tt.wantErr {
				t.Errorf("validateSignal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	sigMap := signal.SignalMap
	for sigStr := range sigMap {
		sig, error := util.ValidateSignal(sigStr)
		assert.NilError(t, error)
		signal := sigMap[sigStr]
		assert.DeepEqual(t, signal, sig)
	}
}

func TestExecuteStopSignal(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "STOPSIGNAL handler test 1",
			dockerfile: `FROM alpine
STOPSIGNAL 15`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.StopSignal, "15")
			},
		},
		{
			name: "STOPSIGNAL handler test 2",
			dockerfile: `FROM alpine
STOPSIGNAL SIGKILL`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.StopSignal, "SIGKILL")
			},
		},
		{
			name: "STOPSIGNAL handler test 3",
			dockerfile: `FROM alpine
STOPSIGNAL       kill`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.StopSignal, "kill")
			},
		},
		{
			name: "STOPSIGNAL handler test 4",
			dockerfile: `FROM alpine
STOPSIGNAL $sig`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr:   true,
			funcCheck: func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "STOPSIGNAL handler test 5",
			dockerfile: `FROM alpine
ARG sig
STOPSIGNAL $sig`,
			buildArgs: map[string]string{"sig": "kill"},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.StopSignal, "kill")
			},
		},
		{
			name: "STOPSIGNAL handler test 6",
			dockerfile: `FROM alpine
ARG sig=kill
STOPSIGNAL SIG$sig`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.StopSignal, "SIGkill")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			var retErr error
			for _, cmd := range s.commands {
				if retErr = cmd.cmdExecutor(); retErr != nil {
					break
				}
			}
			if (retErr != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", retErr, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteUser(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "USER handler test 1 - with name",
			dockerfile: `FROM alpine
USER root`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.User, "root")
			},
		},
		{
			name: "USER handler test 2 - with uid",
			dockerfile: `FROM alpine
USER 1000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.User, "1000")
			},
		},
		{
			name: "USER handler test 3 - with name and uid",
			dockerfile: `FROM alpine
USER jack:1000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.User, "jack:1000")
			},
		},
		{
			name: "USER handler test 4",
			dockerfile: `FROM alpine
USER $usr:$gid`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.User, ":")
			},
		},
		{
			name: "USER handler test 5",
			dockerfile: `FROM alpine
ARG usr
ARG gid
USER $usr:$gid`,
			buildArgs: map[string]string{"usr": "jack", "gid": "1000"},
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.User, "jack:1000")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			var retErr error
			for _, cmd := range s.commands {
				if retErr = cmd.cmdExecutor(); retErr != nil {
					break
				}
			}
			if (retErr != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", retErr, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteExpose(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "EXPOSE handler test 1 - with name",
			dockerfile: `FROM alpine
EXPOSE 8080`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"8080/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 2 - with proto",
			dockerfile: `FROM alpine
EXPOSE 80/TCP`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"80/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 3 - invalid proto",
			dockerfile: `FROM alpine
EXPOSE 80/tcpp`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr:   true,
			funcCheck: func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "EXPOSE handler test 4 - multiple ports",
			dockerfile: `FROM alpine
EXPOSE 80/TCP 3000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"80/tcp":   {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 5 - with empty proto",
			dockerfile: `FROM alpine
EXPOSE 80/`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"80/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 6 - with valid para",
			dockerfile: `FROM alpine
ARG port=80/TCP
EXPOSE $port 3000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"80/tcp":   {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 7 - para with empty proto",
			dockerfile: `FROM alpine
ARG port=80/
EXPOSE $port 3000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"80/tcp":   {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 8 - with invalid para",
			dockerfile: `FROM alpine
ARG port=80/TCPTCP
EXPOSE $port 3000`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			wantErr:   true,
			funcCheck: func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "EXPOSE handler test 9 - with multiple para",
			dockerfile: `FROM alpine
ARG port=8080
ARG port2=3000
ARG proto=udp
EXPOSE ${port}/${proto} ${port2}`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"8080/udp": {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 10 - with multiple para(with${}) but no proto",
			dockerfile: `FROM alpine
ARG port=8080
ARG port2=3000
EXPOSE ${port}/${proto} ${port2}`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"8080/tcp": {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 11 - with multiple para(with$) but no proto",
			dockerfile: `FROM alpine
ARG port=8080
ARG port2=3000
EXPOSE $port/$proto $port2`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"8080/tcp": {},
					"3000/tcp": {},
				})
			},
		},
		{
			name: "EXPOSE handler test 12 - with valid ranged port",
			dockerfile: `FROM alpine
EXPOSE 3000-5000 8080`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.ExposedPorts, docker.PortSet{
					"3000-5000/tcp": {},
					"8080/tcp":      {},
				})
			},
		},
		{
			name: "EXPOSE handler test 13 - with invalid ranged port",
			dockerfile: `FROM alpine
EXPOSE 300-500-/tcp 808`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {},
			wantErr:   true,
		},
		{
			name: "EXPOSE handler test 14 - with invalid port",
			dockerfile: `FROM alpine
EXPOSE 300/t.cp 808`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {},
			wantErr:   true,
		},
		{
			name: "EXPOSE handler test 15 - with invalid ranged port",
			dockerfile: `FROM alpine
EXPOSE 300-500-/tcp`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {},
			wantErr:   true,
		},
		{
			name: "EXPOSE handler test 16 - with invalid ranged port",
			dockerfile: `FROM alpine
EXPOSE 300-500-800/tcp`,
			buildArgs: make(map[string]string),
			config: &docker.Image{
				V1Image: docker.V1Image{
					Config: &docker.Config{},
				},
			},
			funcCheck: func(t *testing.T, s *stageBuilder) {},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			for _, cmd := range s.commands {
				if err = cmd.cmdExecutor(); err != nil {
					break
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteEnv(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "ENV handler test 1",
			dockerfile: `FROM alpine
ENV test1=aa test2=bb test3=cc`,
			buildArgs: make(map[string]string),
			config:    &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Env, []string{"test1=aa", "test2=bb", "test3=cc"})
			},
		},
		{
			name: "ENV handler test 2",
			dockerfile: `FROM alpine
ENV test1 aa
ENV test2 bb
ENV test3 cc
ENV test1 ac`,
			buildArgs: make(map[string]string),
			config:    &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Env, []string{"test1=ac", "test2=bb", "test3=cc"})
			},
		},
		{
			name: "ENV handler test 3",
			dockerfile: `FROM alpine
ARG arg1
ARG arg2
ARG arg3
ENV $arg1 aa
ENV test2 $arg2
ENV test3 c${arg3}c`,
			buildArgs: map[string]string{"arg1": "a1", "arg2": "a2", "arg3": "a3"},
			config:    &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.Env, []string{"a1=aa", "test2=a2", "test3=ca3c"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			for _, cmd := range s.commands {
				if err = cmd.cmdExecutor(); err != nil {
					break
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestExecuteOnbuild(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		buildArgs  map[string]string
		config     *docker.Image
		wantErr    bool
		funcCheck  func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "ONBUILD handler test 1",
			dockerfile: `FROM alpine
ONBUILD ADD . /app/src
ONBUILD RUN /usr/local/bin/python-build --dir /app/src`,
			buildArgs: make(map[string]string),
			config:    &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.DeepEqual(t, s.docker.Config.OnBuild, []string{"ADD . /app/src", "RUN /usr/local/bin/python-build --dir /app/src"})
			},
		},
		{
			name: "ONBUILD handler test 2",
			dockerfile: `FROM alpine
ONBUILD ADD $src ${dst}
ONBUILD RUN /usr/local/bin/$app --dir $dst`,
			buildArgs: map[string]string{"src": ".", "dst": "/app/src", "app": "python-build"},
			config:    &docker.Image{V1Image: docker.V1Image{Config: &docker.Config{}}},
			funcCheck: func(t *testing.T, s *stageBuilder) {
				// word expansion doesn't allowed in ONBUILD. But those may subcommands may be expand at next building
				assert.DeepEqual(t, s.docker.Config.OnBuild, []string{"ADD $src ${dst}", "RUN /usr/local/bin/$app --dir $dst"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder: &Builder{
					buildOpts:    BuildOptions{BuildArgs: tt.buildArgs},
					reservedArgs: make(map[string]string),
					cliLog:       logger.NewCliLogger(constant.CliLogBufferLen),
					ctx:          context.Background(),
				},
				env:      make(map[string]string),
				rawStage: generateOneRawStage(t, tt.dockerfile),
				docker:   tt.config,
			}
			err := s.analyzeStage(context.Background())
			assert.NilError(t, err)
			for _, cmd := range s.commands {
				if err = cmd.cmdExecutor(); err != nil {
					break
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("cmdExecutor() error: %v, wantErr: %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}
