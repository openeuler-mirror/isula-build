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
// Description: stageBuilder related functions

package dockerfile

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/containers/image/v5/pkg/strslice"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	dockerfile "isula.org/isula-build/builder/dockerfile/parser"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

type stageBuilderOption struct {
	systemContext *types.SystemContext
}

type stageBuilder struct {
	builder      *Builder
	buildOpt     *stageBuilderOption
	rawStage     *parser.Page
	commands     []*cmdBuilder
	position     int
	fromStageIdx int

	localStore *store.Store

	name       string
	imageID    string
	topLayer   string
	mountpoint string
	shellForm  strslice.StrSlice

	// data from provided image
	fromImage   string
	fromImageID string
	env         map[string]string

	container   string
	containerID string

	docker *docker.Image
}

// newStageBuilder new a stage builder
func newStageBuilder(idx int, name string) *stageBuilder {
	s := &stageBuilder{
		position: idx,
		name:     name,
		buildOpt: &stageBuilderOption{
			systemContext: image.GetSystemContext(),
		},
		env: make(map[string]string),
		// default shell form in linux is "/bin/sh -c"
		shellForm: strslice.StrSlice{"/bin/sh", "-c"},
	}

	return s
}

func analyzeArg(b *Builder, line *parser.Line, stageArgs, stageEnvs map[string]string) (map[string]string, error) {
	resolveArg := func(s string) string {
		// priority: --build-arg > ENVs in stage > override-ed ARG in stage.
		// we can't use heading ARG directly, it must be active-ed by stage ARG first
		if v, ok := b.buildOpts.BuildArgs[s]; ok {
			return v
		}
		if v, ok := stageEnvs[s]; ok {
			return v
		}
		if v, ok := stageArgs[s]; ok {
			return v
		}
		return ""
	}

	val, err := dockerfile.ResolveParam(line.Cells[0].Value, false, resolveArg)
	if err != nil {
		b.Logger().Errorf("Word expansion for ARG at line %d failed: %v", line.Begin, err)
		return nil, errors.Wrapf(err, "word expansion for ARG at line %d failed", line.Begin)
	}

	kv := strings.Split(val, "=")
	b.Logger().Debugf("analyzeArg: handling ARG at line %d", line.Begin)

	// priority: --build-arg > override ARG in stage > heading ARG. stageEnv not used here
	// activates the ARG usage with stageArgs
	if _, inBuildArgs := b.buildOpts.BuildArgs[kv[0]]; inBuildArgs {
		stageArgs[kv[0]] = b.buildOpts.BuildArgs[kv[0]]
		// activated! Not an unused arg anymore!
		delete(b.unusedArgs, kv[0])
	} else {
		if len(kv) < 2 && stageArgs[kv[0]] == "" && b.headingArgs[kv[0]] != "" {
			stageArgs[kv[0]] = b.headingArgs[kv[0]]
		}
		if len(kv) >= 2 {
			stageArgs[kv[0]] = kv[1]
		}
	}
	return util.CopyMapStringString(stageArgs), nil
}

func analyzeEnv(line *parser.Line, stageArgs, stageEnvs map[string]string) (map[string]string, error) {
	resolveArg := func(s string) string {
		// priority: ENVs in stage > overrided ARG in stage.
		if v, ok := stageEnvs[s]; ok {
			return v
		}
		if v, ok := stageArgs[s]; ok {
			return v
		}
		return ""
	}

	for _, cell := range line.Cells {
		val, err := dockerfile.ResolveParam(cell.Value, false, resolveArg)
		if err != nil {
			logrus.Errorf("Word expansion for ENV at line %d failed: %v", line.Begin, err)
			return nil, errors.Wrapf(err, "word expansion for ENV at line %d failed", line.Begin)
		}

		const elemNum = 2
		kv := strings.SplitN(val, "=", elemNum)
		logrus.Debugf("AnalyseEnv: handling ENV at line %d", line.Begin)
		if len(kv) < elemNum {
			return nil, errors.Errorf("ENV at line %d has too few arguments", line.Begin)
		}

		stageEnvs[kv[0]] = kv[1]
		if _, ok := stageArgs[kv[0]]; ok {
			delete(stageArgs, kv[0])
			logrus.Infof("AnalyseEnv: handling ENV at line %d, found one key is defined by ARG before, "+
				"use with the newer ENV instead", line.Begin)
		}
	}
	return util.CopyMapStringString(stageEnvs), nil
}

func (s *stageBuilder) analyzeStage(ctx context.Context) error {
	var err error
	// those activated reserved args no needs activates again by ARG in the stage
	stageArgs := util.CopyMapStringString(s.builder.reservedArgs)
	// not copying s.env but named stageEnvs point to it directly, it is because by such it can be updated
	// by following analyzeEnv(), and takes effect to the commands after new ENV commands in this stage
	stageEnvs := s.env

	// loop lines and create CmdBuilders
	for _, line := range s.rawStage.Lines {
		s.builder.Logger().Debugf("Analyzing stage: handling line %d command %s", line.Begin, line.Command)
		cb := newCmdBuilder(ctx, line, s, stageArgs, stageEnvs)

		switch line.Command {
		// From cmd is already pre-processed, we just pass it
		case dockerfile.From:
			continue
		case dockerfile.Arg:
			if cb.args, err = analyzeArg(s.builder, line, stageArgs, stageEnvs); err != nil {
				return err
			}
		case dockerfile.Env:
			if cb.envs, err = analyzeEnv(line, stageArgs, stageEnvs); err != nil {
				return err
			}
		case dockerfile.Healthcheck:
			allowFlags := map[string]bool{"start-period": true, "interval": true, "timeout": true, "retries": true, "attribute": true}
			cb.cmdFlags, cb.cmdArgs = getFlagsAndArgs(line, allowFlags)
		}
		s.commands = append(s.commands, cb)
	}

	return nil
}

func (s *stageBuilder) stageBuild(ctx context.Context) (string, error) {
	var err error

	// 1. prepare for new stage building
	if err = s.prepare(ctx); err != nil {
		return "", err
	}
	s.builder.Logger().Debugf("Created mountpoint %s for stage %s", s.mountpoint, s.name)

	// 2. Loop building for commands
	for _, cmd := range s.commands {
		if err = cmd.cmdExecutor(); err != nil {
			return "", errors.Wrapf(err, "handle command %s failed", cmd.line.Command)
		}
	}

	// 3. commit for new image if needed
	if s.rawStage.NeedCommit {
		if s.imageID, err = s.commit(ctx); err != nil {
			return s.imageID, errors.Wrapf(err, "commit image for stage %s failed", s.name)
		}
	}
	// for only from command in Dockerfile, there is no imageID committed, use fromImageID
	if s.imageID == "" {
		s.imageID = s.fromImageID
	}

	return s.imageID, nil
}

func prepareImage(opt *image.PrepareImageOptions) (*image.Describe, error) {
	if opt.FromImage == "" {
		return nil, errors.New("get empty from image")
	}
	var (
		imgID     string
		topLayID  string
		fromImage types.Image
		si        *storage.Image
		err       error
	)

	if opt.FromImage != noBaseImage {
		// check whether fromImage exists in local store, otherwise pull from registry
		fromImage, si, err = image.ResolveFromImage(opt)
		if err != nil {
			return nil, err
		}
		imgID = si.ID
		topLayID = si.TopLayer
	}

	layer, err := image.GetRWLayerByImageID(imgID, opt.Store)
	if err != nil {
		return nil, err
	}

	return &image.Describe{
		Image:         fromImage,
		ImageID:       imgID,
		TopLayID:      topLayID,
		ContainerDesc: layer,
	}, nil
}

// prepare StageBuilder prepares a RWLayer for stage building, returns the mountpoint and error
func (s *stageBuilder) prepare(ctx context.Context) error {
	if len(s.rawStage.Lines) == 0 {
		return errors.Errorf("empty stage builder %s found", s.rawStage.Name)
	}
	// firstLine in each stage is always FROM command
	firstLine := s.rawStage.Lines[0]
	cmdInfo := fmt.Sprintf("%s %s", firstLine.Command, firstLine.Raw)
	logInfo := fmt.Sprintf("%s %d-%d", firstLine.Command, firstLine.Begin, firstLine.End)
	s.builder.cliLog.StepPrint(cmdInfo)
	logTimer := s.builder.cliLog.StartTimer(logInfo)

	imgDesc, err := prepareImage(&image.PrepareImageOptions{
		Ctx:           ctx,
		FromImage:     s.fromImage,
		SystemContext: s.buildOpt.systemContext,
		Store:         s.localStore,
		Reporter:      s.builder.cliLog,
	})
	s.builder.cliLog.StopTimer(logTimer)
	s.builder.Logger().Debugln(s.builder.cliLog.GetCmdTime(logTimer))
	if err != nil {
		return err
	}
	s.fromImageID = imgDesc.ImageID
	s.topLayer = imgDesc.TopLayID
	s.containerID = imgDesc.ContainerDesc.ContainerID
	s.container = imgDesc.ContainerDesc.ContainerName
	s.mountpoint = imgDesc.ContainerDesc.Mountpoint

	if s.docker, err = image.GenerateFromImageSpec(ctx, imgDesc.Image, image.DockerV2Schema2MediaType); err != nil {
		return err
	}
	if err = s.updateStageBuilder(); err != nil {
		return err
	}

	return s.analyzeStage(ctx)
}

func (s *stageBuilder) updateStageBuilder() error {
	if s.docker.Config == nil {
		return nil
	}

	// extracting ENV from provided image
	if s.docker.Config.Env != nil {
		for _, env := range s.docker.Config.Env {
			const elemNum = 2
			items := strings.SplitN(env, "=", elemNum)
			if len(items) != elemNum {
				s.builder.Logger().Warnf("Get bad env %q from image [%q] for build %q", env, s.fromImage, s.name)
				continue
			}
			s.env[items[0]] = items[1]
		}
	}
	if _, ok := s.env["PATH"]; !ok {
		s.env["PATH"] = defaultPathEnv
	}

	if len(s.docker.Config.OnBuild) == 0 {
		return nil
	}

	// extracting ONBUILD from provided image and insert into stage commands
	var onbuildData []byte
	for _, item := range s.docker.Config.OnBuild {
		// only the Linux architecture is supported currently
		onbuildData = append(onbuildData, []byte(fmt.Sprintf("%s\n", item))...)
	}
	// OnBuild is handled, clean it here so that we can add new ONBUILDs on cmd builder if needed
	s.docker.Config.OnBuild = make([]string, 0)

	p, err := parser.NewParser(parser.DefaultParser)
	if err != nil {
		return errors.Wrap(err, "create parser failed")
	}

	playbook, err := p.Parse(bytes.NewReader(onbuildData), true)
	if err != nil {
		return errors.Wrap(err, "parse dockerfile failed")
	}
	// insert the ONBUILD COMMAND
	rear := append([]*parser.Line{}, s.rawStage.Lines[1:]...)
	s.rawStage.Lines = append(append(s.rawStage.Lines[0:1], playbook.Pages[0].Lines...), rear...)

	return nil
}

// commit commits the state in the last CmdBuilder, return imageID and error to caller
func (s *stageBuilder) commit(ctx context.Context) (string, error) {
	if len(s.commands) == 0 {
		return "", errors.Errorf("nothing can be committed in stage %s", s.name)
	}
	return s.commands[len(s.commands)-1].commit(ctx)
}

// delete cleans up temporary resources which are created during stage building.
func (s *stageBuilder) delete() error {
	if s.containerID == "" {
		return nil
	}

	s.mountpoint = ""

	return s.localStore.CleanContainer(s.containerID)
}
