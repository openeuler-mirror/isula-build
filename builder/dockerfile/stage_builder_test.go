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
// Description: stageBuilder related functions tests

package dockerfile

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/reexec"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
	dockerfile "isula.org/isula-build/builder/dockerfile/parser"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
	testUtil "isula.org/isula-build/util"
)

var (
	localStore store.Store
	rootDir    = "/tmp/isula-build/dockerfile"
)

func init() {
	reexec.Init()
	dataRoot := rootDir + "/data"
	runRoot := rootDir + "/run"
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: dataRoot,
		RunRoot:  runRoot,
	})
	localStore, _ = store.GetStore()
}

func TestMain(m *testing.M) {
	fmt.Println("dockerfile package test begin")
	m.Run()
	fmt.Println("dockerfile package test end")
	clean()
}

func clean() {
	if err := unix.Unmount(rootDir+"/data/overlay", 0); err != nil {
		fmt.Printf("umount dir %s failed: %v\n", rootDir+"/data/overlay", err)
	}

	if err := os.RemoveAll(rootDir); err != nil {
		fmt.Printf("remove test root dir %s failed: %v\n", rootDir, err)
	}
}

func cleanAndSetDefaultStoreOpt(t *testing.T) {
	cleanDefaultStoreOpt(t)
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: fmt.Sprintf("/tmp/isula-build/storage-data-%d/", util.GenRandInt64()),
		RunRoot:  fmt.Sprintf("/tmp/isula-build/storage-run-%d/", util.GenRandInt64()),
	})
	localStore, _ = store.GetStore()
}

func cleanDefaultStoreOpt(t *testing.T) {
	store, err := store.GetStore()
	assert.NilError(t, err)

	driverRoot := store.GraphRoot() + "/overlay"
	os.RemoveAll(driverRoot)
	assert.NilError(t, err)
	err = os.RemoveAll(store.RunRoot())
	assert.NilError(t, err)
	err = unix.Unmount(driverRoot, 0)
	assert.NilError(t, err)
	err = os.RemoveAll(store.GraphRoot())
	assert.NilError(t, err)
	err = os.RemoveAll("/tmp/isula-build")
	assert.NilError(t, err)
}

func getImageID(t *testing.T, s *store.Store) string {
	img, err := s.Images()
	assert.NilError(t, err)
	if len(img) > 0 {
		return img[0].ID
	}
	fmt.Printf("get img failed: %v\n", err)
	return ""
}

func getBuilder() *Builder {
	privateKey, _ := util.GenerateRSAKey(util.DefaultRSAKeySize)

	return &Builder{
		ctx:           context.Background(),
		buildID:       "",
		localStore:    &localStore,
		buildOpts:     BuildOptions{},
		cliLog:        logger.NewCliLogger(constant.CliLogBufferLen),
		playbook:      nil,
		ignores:       nil,
		headingArgs:   make(map[string]string),
		reservedArgs:  make(map[string]string),
		unusedArgs:    make(map[string]string),
		stageBuilders: nil,
		rsaKey:        privateKey,
	}
}

func generateOneRawStage(t *testing.T, content string) *parser.Page {
	p, err := parser.NewParser(parser.DefaultParser)
	assert.NilError(t, err)
	playbook, err := p.Parse(bytes.NewReader([]byte(content)), false)
	assert.NilError(t, err)
	assert.Equal(t, len(playbook.Pages), 1)
	assert.Equal(t, len(playbook.Pages[0].Lines), len(strings.Split(content, "\n")))
	assert.Equal(t, playbook.Pages[0].Lines[0].Command, "FROM")

	return playbook.Pages[0]
}

func getImageDesc() string {
	if runtime.GOARCH == "arm64" {
		return "arm64v8/euleros"
	}
	return "amd64/euleros"
}

func getImageDigest() string {
	if runtime.GOARCH == "arm64" {
		return "arm64v8/euleros@sha256:d9659b9b70aced4e1c9b9442af53bf4af12124e2af47948cb5e1c69d9f578e18"
	}
	return "amd64/euleros@sha256:41cff8f10cae502e9d34e1831f1ed3a21fce195169ef956a7aaa1899dc469c41"
}

func TestPrepareFromImage(t *testing.T) {
	testArgs := testUtil.GetTestingArgs(t)
	if _, ok := testArgs[testUtil.SkipRegTestKey]; ok {
		t.Skipf("skipping test, because of args passed: %s\n", testUtil.SkipRegTestKey)
	}
	if reg, ok := testArgs[testUtil.TestRegistryKey]; ok {
		testUtil.DefaultTestRegistry = reg
	}

	contentJustForFillLog := `FROM busybox
CMD ["sh"]`
	type fields struct {
		buildOpt   *stageBuilderOption
		builder    *Builder
		localStore *store.Store
		rawStage   *parser.Page

		name       string
		imageID    string
		position   int
		topLayer   string
		commands   []*cmdBuilder
		mountpoint string

		// data from provided image
		fromImage   string
		fromImageID string
		container   string
		containerID string
		docker      docker.Image
		env         map[string]string
	}
	type args struct {
		ctx  context.Context
		base string
	}
	tests := []struct {
		depLast   bool
		resetID   bool
		name      string
		fields    fields
		args      args
		wantImgID string
		wantErr   bool
	}{
		{
			name: "empty fromImage",
			fields: fields{
				name:      "stage1",
				builder:   getBuilder(),
				fromImage: "",
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			wantErr: true,
		},
		{
			name: "reg+tag",
			fields: fields{
				name:      "stage2",
				builder:   getBuilder(),
				fromImage: filepath.Join(testUtil.DefaultTestRegistry, getImageDesc()),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			depLast: true,
			resetID: true,
			name:    "image id reuse",
			fields: fields{
				name:    "stage3",
				builder: getBuilder(),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			depLast: true,
			name:    "reg+tag reuse",
			fields: fields{
				name:      "stage4",
				builder:   getBuilder(),
				fromImage: filepath.Join(testUtil.DefaultTestRegistry, getImageDesc()),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},

		{
			depLast: true,
			name:    "digest reuse",
			fields: fields{
				name:    "stage5",
				builder: getBuilder(),
				// NOTE:If the digest changes, the test case fails to be executed.
				fromImage: filepath.Join(testUtil.DefaultTestRegistry, getImageDigest()),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "error digest",
			fields: fields{
				name:      "stage6",
				builder:   getBuilder(),
				fromImage: filepath.Join(testUtil.DefaultTestRegistry, "busybox@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
		{
			name: "error tag",
			fields: fields{
				name:      "stage7",
				builder:   getBuilder(),
				fromImage: filepath.Join(testUtil.DefaultTestRegistry, "busybox:aaaaaaaaaaa"),
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
		{
			name: "scratch",
			fields: fields{
				name:      "stage8",
				builder:   getBuilder(),
				fromImage: "scratch",
				buildOpt: &stageBuilderOption{systemContext: &types.SystemContext{
					SignaturePolicyPath:      constant.SignaturePolicyPath,
					SystemRegistriesConfPath: constant.RegistryConfigPath,
					RegistriesDirPath:        constant.RegistryDirPath,
				}},
				rawStage: generateOneRawStage(t, contentJustForFillLog),
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder:     tt.fields.builder,
				buildOpt:    tt.fields.buildOpt,
				name:        tt.fields.name,
				position:    tt.fields.position,
				fromImage:   tt.fields.fromImage,
				fromImageID: tt.fields.fromImageID,
				topLayer:    tt.fields.topLayer,
				commands:    tt.fields.commands,
				mountpoint:  tt.fields.mountpoint,
				container:   tt.fields.container,
				containerID: tt.fields.containerID,
				imageID:     tt.fields.imageID,
				rawStage:    tt.fields.rawStage,
				env:         tt.fields.env,
				localStore:  &localStore,
			}
			if !tt.depLast {
				cleanAndSetDefaultStoreOpt(t)
			}

			if tt.depLast && tt.resetID {
				s.fromImage = getImageID(t, s.localStore)
			}
			s.env = make(map[string]string)
			err := s.prepare(tt.args.ctx)
			if s.mountpoint != "" {
				_, err := s.localStore.Unmount(s.containerID, false)
				assert.NilError(t, err)
			}
			logrus.Infof("get mount point %q", s.mountpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("StageBuild() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
	cleanDefaultStoreOpt(t)
}

func TestUpdateStageBuilder(t *testing.T) {
	type fields struct {
		builder     *Builder
		localStore  *store.Store
		buildOpt    *stageBuilderOption
		name        string
		imageID     string
		position    int
		topLayer    string
		commands    []*cmdBuilder
		mountpoint  string
		fromImage   string
		fromImageID string
		container   string
		containerID string
		docker      *docker.Image
		env         map[string]string
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		funcCheck func(t *testing.T, s *stageBuilder)
	}{
		{
			name: "normal test",
			fields: fields{
				docker: &docker.Image{
					Parent: docker.ID("aaa"),
					V1Image: docker.V1Image{
						Config: &docker.Config{
							OnBuild: []string{
								"ENV onbuild=true",
								"RUN mkdir -p /home/onbuild",
								"RUN touch /home/onbuild/onbuild.txt",
								"COPY /bin/ls /home/onbuild/",
								"ARG aaa=onbuild_arg",
								"ARG aaa-onbuild=onbuild_arg",
							},
						},
					},
				},
				env: make(map[string]string),
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				assert.Equal(t, len(s.rawStage.Lines), 10)
				assert.Equal(t, s.rawStage.Lines[0].Command, "FROM")
				assert.Equal(t, s.rawStage.Lines[1].Command, "ENV")
				assert.Equal(t, s.rawStage.Lines[1].Raw, "onbuild=true")
				assert.Equal(t, s.rawStage.Lines[2].Command, "RUN")
				assert.Equal(t, s.rawStage.Lines[3].Command, "RUN")
				assert.Equal(t, s.rawStage.Lines[4].Command, "COPY")
				assert.Equal(t, s.rawStage.Lines[5].Command, "ARG")
				assert.Equal(t, s.rawStage.Lines[5].Raw, "aaa=onbuild_arg")
				assert.Equal(t, s.rawStage.Lines[6].Command, "ARG")
				assert.Equal(t, s.rawStage.Lines[7].Command, "ENV")
				assert.Equal(t, s.rawStage.Lines[8].Command, "COPY")
				assert.Equal(t, s.rawStage.Lines[9].Command, "CMD")
			},
		},
		{
			name: "nil Config test",
			fields: fields{
				docker: &docker.Image{
					Parent: docker.ID("aaa"),
					V1Image: docker.V1Image{
						Config: nil,
					},
				},
				env: make(map[string]string),
			},
			wantErr:   false,
			funcCheck: func(t *testing.T, s *stageBuilder) {},
		},
		{
			name: "extracting ENV test",
			fields: fields{
				docker: &docker.Image{
					Parent: docker.ID("aaa"),
					V1Image: docker.V1Image{
						Config: &docker.Config{
							Env: []string{"foo1=bar1", "foo2=bar2"},
						},
					},
				},
				env: make(map[string]string),
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				// "PATH" and other 2 envs
				assert.Equal(t, len(s.env), 3)
				assert.DeepEqual(t, s.env, map[string]string{
					"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"foo1": "bar1",
					"foo2": "bar2"})
			},
		},
		{
			name: "extracting ENV with bad format test",
			fields: fields{
				docker: &docker.Image{
					Parent: docker.ID("aaa"),
					V1Image: docker.V1Image{
						Config: &docker.Config{
							Env: []string{"foo1=bar1", "foo2"},
						},
					},
				},
				env:     make(map[string]string),
				builder: getBuilder(),
			},
			wantErr: false,
			funcCheck: func(t *testing.T, s *stageBuilder) {
				// "PATH" and other 2 envs
				assert.Equal(t, len(s.env), 2)
				assert.DeepEqual(t, s.env, map[string]string{
					"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"foo1": "bar1"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &stageBuilder{
				builder:     tt.fields.builder,
				localStore:  tt.fields.localStore,
				buildOpt:    tt.fields.buildOpt,
				name:        tt.fields.name,
				imageID:     tt.fields.imageID,
				position:    tt.fields.position,
				topLayer:    tt.fields.topLayer,
				commands:    tt.fields.commands,
				mountpoint:  tt.fields.mountpoint,
				fromImage:   tt.fields.fromImage,
				fromImageID: tt.fields.fromImageID,
				container:   tt.fields.container,
				containerID: tt.fields.containerID,
				docker:      tt.fields.docker,
				env:         tt.fields.env,
			}
			content := `FROM alpine AS uuid
ENV myenv=aaa
COPY uuid /src/data
CMD ["/bin/sh", "-c", "sleep 1000"]`
			s.rawStage = generateOneRawStage(t, content)
			if err := s.updateStageBuilder(); (err != nil) != tt.wantErr {
				t.Errorf("updateStageBuilder() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.funcCheck(t, s)
		})
	}
}

func TestAnalyseArgAndEnv(t *testing.T) {
	type args struct {
		line      *parser.Line
		stageArgs map[string]string
		stageEnvs map[string]string
	}
	tests := []struct {
		name      string
		args      args
		content   string
		buildArgs map[string]string
		wantArgs  map[string]string
		wantEnvs  map[string]string
		wantErr   bool
	}{
		{
			name: "ARG and ENV scope testing",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1", "arg-key2": "arg-value2"},
				stageEnvs: map[string]string{"env-key1": "env-value1", "env-key2": "env-value2"},
			},
			content: `ARG testArgs=global
FROM alpine AS stage1
ARG testArg2=foo
ENV testEnv=bar`,
			buildArgs: map[string]string{"unusedArg1": "arg1", "testArg2": "arg2", "unnamedArg": "arg3"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "arg-key2": "arg-value2", "testArg2": "arg2"},
			wantEnvs:  map[string]string{"env-key1": "env-value1", "env-key2": "env-value2", "testEnv": "bar"},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 2 - without build-arg",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{"env-key1": "env-value1"},
			},
			content: `FROM alpine
			ENV testEnv=env2
			ARG testArg=$testEnv`,
			buildArgs: map[string]string{},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "testArg": "env2"},
			wantEnvs:  map[string]string{"env-key1": "env-value1", "testEnv": "env2"},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 2 - with build-arg",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{"env-key1": "env-value1"},
			},
			content: `FROM alpine
			ENV testEnv=env2
			ARG testArg=$testEnv`,
			buildArgs: map[string]string{"testArg": "arg2"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "testArg": "arg2"},
			wantEnvs:  map[string]string{"env-key1": "env-value1", "testEnv": "env2"},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 3 - with heading arg 1",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{},
			},
			content: `ARG harg=$testharg
			FROM alpine
			ARG harg
			ARG testArg=$harg`,
			buildArgs: map[string]string{"testharg": "arg2"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "testArg": ""},
			wantEnvs:  map[string]string{},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 3 - with heading arg 2",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{},
			},
			content: `ARG testharg
			ARG harg=$testharg
			FROM alpine
			ARG harg
			ARG testArg=$harg`,
			buildArgs: map[string]string{"testharg": "arg2"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "harg": "arg2", "testArg": "arg2"},
			wantEnvs:  map[string]string{},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 3 - with heading arg 3",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{},
			},
			content: `ARG testharg
			ARG harg=$testharg
			FROM alpine
			ARG harg
			ARG testArg=$harg`,
			buildArgs: map[string]string{},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "testArg": ""},
			wantEnvs:  map[string]string{},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 4",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{"testEnv": "env-value1"},
			},
			content: `FROM alpine
			ARG env1
			ARG env2
			ARG env3
			ARG env4
			ENV testEnv=$env1 testEnv2=${env2} testEnv3=$env3 testEnv4=${env4}nice`,
			buildArgs: map[string]string{"env1": "e1", "env2": "e2", "env3": "e3", "env4": "e4"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "env1": "e1", "env2": "e2", "env3": "e3", "env4": "e4"},
			wantEnvs:  map[string]string{"testEnv": "e1", "testEnv2": "e2", "testEnv3": "e3", "testEnv4": "e4nice"},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 5",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{"testEnv": "env-value1"},
			},
			content: `FROM alpine
			ARG env1
			ARG env2
			ARG env3
			ARG env4
			ENV testEnv $env1
			ENV testEnv2 ${env2}
			ENV testEnv3 $env3
			ENV testEnv4 ${env4}nice
			ENV testEnv5 t${arg-key1}`,
			buildArgs: map[string]string{"env1": "e1", "env2": "e2", "env3": "e3", "env4": "e4"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "env1": "e1", "env2": "e2", "env3": "e3", "env4": "e4"},
			wantEnvs:  map[string]string{"testEnv": "e1", "testEnv2": "e2", "testEnv3": "e3", "testEnv4": "e4nice", "testEnv5": "targ-value1"},
			wantErr:   false,
		},
		{
			name: "ARG and ENV scope testing 6 - mixed ENV and ARG params",
			args: args{
				line:      nil,
				stageArgs: map[string]string{"arg-key1": "arg-value1"},
				stageEnvs: map[string]string{"testEnv": "env-value1"},
			},
			content: `FROM alpine
			ARG env1
			ENV testEnv $env1
			ARG testEnv
			ARG env3
			ENV testEnv $env3
			ARG testEnv3`,
			buildArgs: map[string]string{"env1": "e1", "env3": "e3"},
			wantArgs:  map[string]string{"arg-key1": "arg-value1", "env1": "e1", "env3": "e3"},
			wantEnvs:  map[string]string{"testEnv": "e3"},
			wantErr:   false,
		},
		{
			name: "ENV testing 7 - ENV contains some =",
			args: args{
				line:      nil,
				stageArgs: map[string]string{},
				stageEnvs: map[string]string{},
			},
			content: `FROM alpine
			ENV testEnv=env1=5
			ENV testEnv2 env2=5`,
			buildArgs: map[string]string{},
			wantArgs:  map[string]string{},
			wantEnvs:  map[string]string{"testEnv": "env1=5", "testEnv2": "env2=5"},
			wantErr:   false,
		},
		{
			name: "resolve param failed testing",
			args: args{
				line:      nil,
				stageArgs: map[string]string{},
				stageEnvs: map[string]string{},
			},
			content: `FROM alpine
			ARG testEnv=${foo)
            ENV testEnv=${foo)`,
			buildArgs: map[string]string{},
			wantArgs:  nil,
			wantEnvs:  nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				buildOpts: BuildOptions{
					File:      tt.content,
					BuildArgs: tt.buildArgs,
				},
				cliLog: logger.NewCliLogger(constant.CliLogBufferLen),
				ctx:    context.Background(),
			}

			err := b.parseFiles()
			assert.NilError(t, err)
			err = b.newStageBuilders()
			assert.NilError(t, err)

			var lineArgs = make(map[string]string)
			var lineEnvs = make(map[string]string)
			for _, sb := range b.stageBuilders {
				for _, line := range sb.rawStage.Lines {
					tt.args.line = line
					switch line.Command {
					case dockerfile.Arg:
						lineArgs, err = analyzeArg(b, tt.args.line, tt.args.stageArgs, tt.args.stageEnvs)
						if tt.wantErr == false {
							assert.NilError(t, err)
						}
					case dockerfile.Env:
						lineEnvs, err = analyzeEnv(tt.args.line, tt.args.stageArgs, tt.args.stageEnvs)
						if tt.wantErr == false {
							assert.NilError(t, err)
						}
					default:
					}
				}
			}
			assert.DeepEqual(t, lineArgs, tt.wantArgs)
			assert.DeepEqual(t, lineEnvs, tt.wantEnvs)
		})
	}
}
