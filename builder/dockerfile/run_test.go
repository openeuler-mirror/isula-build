// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2020-03-20
// Description: RUN command related functions tests

package dockerfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"gotest.tools/assert"

	constant "isula.org/isula-build"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/pkg/parser"
)

func TestCmdBuilderRun(t *testing.T) {
	sb := &stageBuilder{
		builder: &Builder{
			ctx: context.Background(),
			buildOpts: BuildOptions{
				ProxyFlag: true,
			},
			cliLog:      logger.NewCliLogger(constant.CliLogBufferLen),
			runtimePath: "abc",
		},
		mountpoint: "/isulatest/mountpoint",
		docker: &docker.Image{
			V1Image: docker.V1Image{
				Config: &docker.Config{
					WorkingDir: "/isulatest/workdir",
				},
			},
		},
	}

	cb := newCmdBuilder(context.Background(), &parser.Line{}, sb, nil, nil)
	command := []string{"aa/bin/sh", "-c", "ls"}

	err := cb.Run(command)
	assert.ErrorContains(t, err, "error running container")
}

func TestSetupBundlePath(t *testing.T) {
	bundlePath, err := setupBundlePath("", "testContainer")
	assert.NilError(t, err)
	defer os.RemoveAll(bundlePath)

	_, err = os.Stat(bundlePath)
	assert.NilError(t, err)
}

func TestSetupBindFiles(t *testing.T) {
	bundlePath, err := setupBundlePath("", "testContainer")
	assert.NilError(t, err)
	defer os.RemoveAll(bundlePath)

	bindFiles, err := setupBindFiles(bundlePath)
	assert.NilError(t, err)
	assert.Equal(t, bindFiles["/etc/hosts"], filepath.Join(bundlePath, "hosts"))
	assert.Equal(t, bindFiles["/etc/resolv.conf"], filepath.Join(bundlePath, "resolv.conf"))

	_, err = os.Stat(filepath.Join(bundlePath, "hosts"))
	assert.NilError(t, err)

	_, err = os.Stat(filepath.Join(bundlePath, "resolv.conf"))
	assert.NilError(t, err)
}

func TestSetupMounts(t *testing.T) {
	bundlePath, err := setupBundlePath("", "testContainer")
	assert.NilError(t, err)
	defer os.RemoveAll(bundlePath)

	gp, err := generate.New("linux")
	assert.NilError(t, err)
	g := &gp
	spec := g.Config

	oriLen := len(spec.Mounts)
	bindFiles, err := setupBindFiles(bundlePath)
	assert.NilError(t, err)

	setupMounts(spec, bindFiles)
	assert.Equal(t, len(spec.Mounts), oriLen+3)
}

func TestSetupMountsDuplicate(t *testing.T) {
	bundlePath, err := setupBundlePath("", "testContainer")
	assert.NilError(t, err)
	defer os.RemoveAll(bundlePath)

	gp, err := generate.New("linux")
	assert.NilError(t, err)
	g := &gp
	spec := g.Config

	oriLen := len(spec.Mounts)
	bindFiles, err := setupBindFiles(bundlePath)
	assert.NilError(t, err)

	spec.Mounts = append(spec.Mounts, specs.Mount{
		Source:      "/test/hosts",
		Destination: "/etc/hosts",
		Type:        "bind",
		Options:     []string{"rbind", "ro"},
	})
	assert.Equal(t, len(spec.Mounts), oriLen+1)

	setupMounts(spec, bindFiles)
	assert.Equal(t, len(spec.Mounts), oriLen+3)
}

func TestSetupRuntimeSpec(t *testing.T) {
	sb := &stageBuilder{
		builder: &Builder{
			buildOpts: BuildOptions{
				ProxyFlag: true,
			},
		},
		mountpoint: "/isulatest/mountpoint",
		docker: &docker.Image{
			V1Image: docker.V1Image{
				Config: &docker.Config{
					WorkingDir: "/isulatest/workdir",
				},
			},
		},
	}
	stageArgs := make(map[string]string)
	stageArgs["arg1"] = "arg1"
	stageArgs["arg2"] = "arg2"

	stageEnvs := make(map[string]string)
	stageEnvs["PATH"] = "/usr/bin"
	stageEnvs["arg1"] = "env1"

	cb := newCmdBuilder(context.Background(), &parser.Line{}, sb, stageArgs, stageEnvs)
	command := []string{"/bin/sh", "-c", "ls"}

	spec, err := cb.setupRuntimeSpec(command)
	assert.NilError(t, err)
	assert.Equal(t, spec.Process.Args[0], "/bin/sh")
	assert.Equal(t, spec.Process.Terminal, false)
	assert.Equal(t, spec.Root.Path, "/isulatest/mountpoint")
	assert.Equal(t, len(spec.Linux.Namespaces), 4)
	assert.Equal(t, contains(spec.Process.Env, "PATH=/usr/bin"), true)
	assert.Equal(t, contains(spec.Process.Env, "arg1=env1"), true)
	assert.Equal(t, contains(spec.Process.Env, "arg2=arg2"), true)
	assert.Equal(t, spec.Process.Cwd, "/isulatest/workdir")
	assert.Equal(t, spec.Linux.MaskedPaths[0], "/proc/acpi")
	assert.Equal(t, spec.Linux.ReadonlyPaths[0], "/proc/asound")
}

func contains(envs []string, env string) bool {
	for _, i := range envs {
		if i == env {
			return true
		}
	}
	return false
}
