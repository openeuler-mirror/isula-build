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
// Description: cmdBuilder related functions

package dockerfile

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	dockerfile "isula.org/isula-build/builder/dockerfile/parser"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/util"
)

var (
	cmdExecutors map[string]func(cb *cmdBuilder) error
)

func init() {
	cmdExecutors = map[string]func(cb *cmdBuilder) error{
		dockerfile.Add:         executeAdd,
		dockerfile.Arg:         executeNoop,
		dockerfile.Copy:        executeCopy,
		dockerfile.Cmd:         executeCmd,
		dockerfile.Entrypoint:  executeEntrypoint,
		dockerfile.Env:         executeEnv,
		dockerfile.Expose:      executeExpose,
		dockerfile.Healthcheck: executeHealthCheck,
		dockerfile.Label:       executeLabel,
		dockerfile.Maintainer:  executeMaintainer,
		dockerfile.OnBuild:     executeOnbuild,
		dockerfile.Run:         executeRun,
		dockerfile.Shell:       executeShell,
		dockerfile.Volume:      executeVolume,
		dockerfile.StopSignal:  executeStopSignal,
		dockerfile.User:        executeUser,
		dockerfile.WorkDir:     executeWorkDir,
	}
}

type cmdBuilder struct {
	// stageBuilder of this command
	stage *stageBuilder

	// line contents parsed by parser
	line *parser.Line

	// context of current builder
	ctx context.Context

	// envs is from stage builder envs
	envs map[string]string

	// args passed parameters from build-time
	args map[string]string

	// the args of the commands
	cmdArgs []string

	// flags for this command
	cmdFlags map[string]string
}

// NewCmdBuilder init a CmdBuilder
func newCmdBuilder(ctx context.Context, line *parser.Line, s *stageBuilder, stageArgs, stageEnvs map[string]string) *cmdBuilder {
	return &cmdBuilder{
		ctx:   ctx,
		stage: s,
		args:  util.CopyMapStringString(stageArgs),
		envs:  util.CopyMapStringString(stageEnvs),
		line:  line,
	}
}

// allowWordExpand these commands supports word expansion.
// note: ARG and ENV are expanded before by stage builder
var allowWordExpand = map[string]bool{
	dockerfile.Add:        true,
	dockerfile.Copy:       true,
	dockerfile.Expose:     true,
	dockerfile.Label:      true,
	dockerfile.StopSignal: true,
	dockerfile.User:       true,
	dockerfile.Volume:     true,
	dockerfile.WorkDir:    true,
}

func (c *cmdBuilder) cmdExecutor() error {
	var err error
	if _, ok := cmdExecutors[c.line.Command]; !ok {
		return errors.Errorf("command [%s %s] not supported for executing, please check!", c.line.Command, c.line.Raw)
	}

	cmdInfo := fmt.Sprintf("%s %s", c.line.Command, c.line.Raw)
	logInfo := fmt.Sprintf("%s %d-%d", c.line.Command, c.line.Begin, c.line.End)
	c.stage.builder.cliLog.StepPrint(cmdInfo)
	logTimer := c.stage.builder.cliLog.StartTimer(logInfo)

	if allowWordExpand[c.line.Command] {
		if err = c.wordExpansion(); err != nil {
			return err
		}
	}

	c.stage.builder.Logger().Infof("Executing line %d command %s", c.line.Begin, c.line.Command)
	err = cmdExecutors[c.line.Command](c)

	c.stage.builder.cliLog.StopTimer(logTimer)
	c.stage.builder.Logger().Debugln(c.stage.builder.cliLog.GetCmdTime(logTimer))
	return err
}

func (c *cmdBuilder) wordExpansion() error {
	resolveArg := func(s string) string {
		c.stage.builder.Logger().Debugf("Resolve Param handling for %s", s)
		if v, ok := c.envs[s]; ok {
			return v
		}
		if v, ok := c.args[s]; ok {
			return v
		}
		return ""
	}

	for i, cell := range c.line.Cells {
		val, err := dockerfile.ResolveParam(cell.Value, false, resolveArg)
		if err != nil {
			c.stage.builder.Logger().
				Errorf("Word expansion for line %d command %s failed: %v", c.line.Begin, c.line.Command, err)
			return errors.Wrapf(err, "word expansion for %s at line %d failed", c.line.Command, c.line.Begin)
		}
		c.line.Cells[i].Value = val
	}

	return nil
}

// FROM/ARG were pre-analyzed by stage builder before
// noop here just for step printing
func executeNoop(cb *cmdBuilder) error {
	return nil
}

func executeCopy(cb *cmdBuilder) error {
	return cb.executeAddAndCopy(false)
}

func executeAdd(cb *cmdBuilder) error {
	return cb.executeAddAndCopy(true)
}

func executeHealthCheck(cb *cmdBuilder) error {
	// the default value is referenced  from https://docs.docker.com/engine/reference/builder/#healthcheck
	var (
		allFlags = map[string]string{
			dockerfile.HealthCheckStartPeriod: "0s",
			dockerfile.HealthCheckInterval:    "30s",
			dockerfile.HealthCheckTimeout:     "30s",
			dockerfile.HealthCheckRetries:     "3",
		}
		durationFlags = map[string]time.Duration{
			dockerfile.HealthCheckStartPeriod: 0,
			dockerfile.HealthCheckInterval:    30 * time.Second,
			dockerfile.HealthCheckTimeout:     30 * time.Second,
		}
	)

	// cb.cmdArgs has at lease 1 arg, which has already checked at parser
	checkType := cb.cmdArgs[0]
	// process NONE type
	if strings.ToUpper(checkType) == healthCheckTestDisable {
		cb.stage.docker.Config.Healthcheck = &docker.HealthConfig{
			Test: []string{healthCheckTestDisable},
		}
		return nil
	}

	const minCmdArgs = 2
	if len(cb.cmdArgs) < minCmdArgs {
		return errors.Errorf("args invalid: %v, HEALTHCHECK must have at least two args", cb.cmdArgs)
	}

	argv := cb.cmdArgs[1:]
	healthcheck := docker.HealthConfig{}
	if cb.line.IsJSONArgs() {
		healthcheck.Test = append(healthcheck.Test, checkType)
		healthcheck.Test = append(healthcheck.Test, argv...)
	} else {
		// if not json cmd, rewrite checkType to 'CMD-SHELL'
		checkType = healthCheckTestTypeShell
		healthcheck.Test = []string{checkType, strings.Join(argv, " ")}
	}

	for flag, defaultValue := range allFlags {
		if _, ok := cb.cmdFlags[flag]; !ok {
			cb.cmdFlags[flag] = defaultValue
		}
	}

	for flag := range durationFlags {
		value := cb.cmdFlags[flag]
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		durationFlags[flag] = d
	}
	healthcheck.StartPeriod = durationFlags["start-period"]
	healthcheck.Interval = durationFlags["interval"]
	healthcheck.Timeout = durationFlags["timeout"]

	retries, err := strconv.Atoi(cb.cmdFlags["retries"])
	if err != nil {
		return err
	}
	healthcheck.Retries = retries
	cb.stage.docker.Config.Healthcheck = &healthcheck

	return nil
}

func executeCmd(cb *cmdBuilder) error {
	var cmdLine []string
	if cb.line.IsJSONArgs() {
		_, cmdLine = getFlagsAndArgs(cb.line, make(map[string]bool))
	} else {
		cmdLine = append(cb.stage.shellForm, cb.line.Cells[0].Value) // nolint:gocritic
	}

	cb.stage.docker.Config.Cmd = cmdLine
	return nil
}

func executeShell(cb *cmdBuilder) error {
	_, cmdLine := getFlagsAndArgs(cb.line, make(map[string]bool))

	cb.stage.shellForm = cmdLine
	cb.stage.docker.Config.Shell = cmdLine
	return nil
}

func executeRun(cb *cmdBuilder) error {
	var cmdLine []string
	if cb.line.IsJSONArgs() {
		_, cmdLine = getFlagsAndArgs(cb.line, make(map[string]bool))
	} else {
		cmdLine = append(cb.stage.shellForm, cb.line.Cells[0].Value) // nolint:gocritic
	}

	return cb.Run(cmdLine)
}

func executeEntrypoint(cb *cmdBuilder) error {
	var entrypoint []string
	if cb.line.IsJSONArgs() {
		_, entrypoint = getFlagsAndArgs(cb.line, make(map[string]bool))
	} else {
		entrypoint = append(cb.stage.shellForm, cb.line.Cells[0].Value) // nolint:gocritic
	}

	cb.stage.docker.Config.Entrypoint = entrypoint
	return nil
}

// ENV was pre-analyzed by stage builder before
// here just add to config
func executeEnv(cb *cmdBuilder) error {
	var envs []string
	for k, v := range cb.envs {
		envs = append(envs, k+"="+v)
	}
	sort.Strings(envs)
	cb.stage.docker.Config.Env = envs
	return nil
}

// ONBUILD was pre-analyzed by stage builder before
// here just add to config
func executeOnbuild(cb *cmdBuilder) error {
	cb.stage.docker.Config.OnBuild = append(cb.stage.docker.Config.OnBuild, cb.line.Raw)
	return nil
}

func executeVolume(cb *cmdBuilder) error {
	if cb.stage.docker.Config.Volumes == nil {
		cb.stage.docker.Config.Volumes = make(map[string]struct{}, len(cb.line.Cells))
	}
	for _, cell := range cb.line.Cells {
		if cell.Value != "" {
			cb.stage.docker.Config.Volumes[cell.Value] = struct{}{}
		}
	}
	if len(cb.stage.docker.Config.Volumes) == 0 {
		return errors.New("no specified dirs in VOLUME")
	}
	return nil
}

func executeLabel(cb *cmdBuilder) error {
	if cb.stage.docker.Config.Labels == nil {
		cb.stage.docker.Config.Labels = make(map[string]string, len(cb.line.Cells))
	}
	for _, cell := range cb.line.Cells {
		kv := strings.Split(cell.Value, "=")
		if len(kv) < 2 {
			return errors.Errorf("%q is not a valid label", cell.Value)
		}
		cb.stage.docker.Config.Labels[kv[0]] = kv[1]
	}
	return nil
}

func executeWorkDir(cb *cmdBuilder) error {
	var (
		origDir = cb.line.Cells[0].Value
		workDir = origDir
	)

	if !path.IsAbs(workDir) {
		workDir = path.Join(string(os.PathSeparator), cb.stage.docker.Config.WorkingDir, workDir)
	}

	p, err := securejoin.SecureJoin(cb.stage.mountpoint, workDir)
	if err != nil {
		return errors.Wrapf(err, "failed to secure join workdir %q", origDir)
	}

	_, err = os.Stat(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "invalid container path %q", origDir)
		}
		// this workdir is created in rootfs, so the dir perm mode should be shared
		if err = os.MkdirAll(p, constant.DefaultSharedDirMode); err != nil {
			return errors.Wrapf(err, "failed to create container path %q", origDir)
		}
	}

	cb.stage.docker.Config.WorkingDir = workDir
	return nil
}

func executeMaintainer(cb *cmdBuilder) error {
	maintainer := cb.line.Cells[0].Value
	cb.stage.docker.Author = maintainer
	return nil
}

func executeStopSignal(cb *cmdBuilder) error {
	if _, err := util.ValidateSignal(cb.line.Cells[0].Value); err != nil {
		return err
	}

	cb.stage.docker.Config.StopSignal = cb.line.Cells[0].Value
	return nil
}

func executeUser(cb *cmdBuilder) error {
	user := cb.line.Cells[0].Value
	cb.stage.docker.Config.User = user
	return nil
}

func executeExpose(cb *cmdBuilder) error {
	if cb.stage.docker.Config.ExposedPorts == nil {
		cb.stage.docker.Config.ExposedPorts = make(docker.PortSet, len(cb.line.Cells))
	}
	for _, cell := range cb.line.Cells {
		p, err := util.PortSet(cell.Value)
		if err != nil {
			return err
		}
		cb.stage.docker.Config.ExposedPorts[docker.Port(p)] = struct{}{}
	}
	return nil
}
