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
// Description: RUN command related functions

package dockerfile

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/ioutils"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/docker/libnetwork/resolvconf"
	"github.com/docker/libnetwork/types"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/runner"
	"isula.org/isula-build/util"
)

// Run executes the specified command in rootfs
func (c *cmdBuilder) Run(command []string) error {
	// create bundle directory for container running
	bundlePath, err := setupBundlePath(c.stage.builder.runDir, c.stage.container)
	if err != nil {
		return err
	}
	defer func() {
		if err2 := os.RemoveAll(bundlePath); err2 != nil {
			c.stage.builder.Logger().Errorf("Removing %q failed: %v", bundlePath, err2)
		}
	}()

	// generate and set runtime spec
	spec, err := c.setupRuntimeSpec(command)
	if err != nil {
		return err
	}

	// setup bind files needed by container running
	bindFiles, err := setupBindFiles(bundlePath)
	if err != nil {
		return err
	}

	// setup all mounts
	setupMounts(spec, bindFiles)

	return runner.NewOCIRunner(&runner.OCIRunOpts{
		Ctx:         c.stage.builder.ctx,
		Spec:        spec,
		RuntimePath: c.stage.builder.runtimePath,
		BundlePath:  bundlePath,
		NoPivot:     false,
		Output:      c.stage.builder.cliLog,
	}).Run()
}

func setupBundlePath(runDir, containerName string) (string, error) {
	bp, err := ioutil.TempDir(runDir, containerName+"-")
	if err != nil {
		return "", errors.Errorf("runner: create directory for RUN error: %v", err)
	}
	bundlePath, err := filepath.EvalSymlinks(bp)
	if err != nil {
		if rerr := os.RemoveAll(bp); rerr != nil {
			logrus.Errorf("Removing %q failed: %v", bp, rerr)
		}
		return "", errors.Errorf("runner: evaluate %s for symbolic links err: %v", bp, err)
	}

	logrus.Debugf("Using %s to hold container bundle data", bp)
	return bundlePath, nil
}

func (c *cmdBuilder) setupRuntimeSpec(command []string) (*specs.Spec, error) {
	// initialize runtime spec
	g, err := generate.New("linux")
	if err != nil {
		return nil, errors.Errorf("initialize runtime spec error: %v", err)
	}

	// set specific runtime spec config
	user := c.stage.docker.Config.User
	if user != "" {
		pair, err := util.GetChownOptions(user, c.stage.mountpoint)
		if err != nil {
			return nil, err
		}
		g.SetProcessUID(uint32(pair.UID))
		g.SetProcessGID(uint32(pair.GID))
		g.SetProcessUsername(c.stage.docker.Config.User)
	}
	g.RemoveHostname()
	g.SetProcessArgs(command)
	g.SetProcessTerminal(false)
	g.SetRootPath(c.stage.mountpoint)
	if err = g.RemoveLinuxNamespace(string(specs.NetworkNamespace)); err != nil {
		return nil, err
	}

	if c.stage.builder.buildOpts.ProxyFlag {
		for envProxy := range constant.ReservedArgs {
			if envProxyValue := os.Getenv(envProxy); envProxyValue != "" {
				g.AddProcessEnv(envProxy, envProxyValue)
			}
		}
	}

	for cbArg, cbArgVal := range c.args {
		g.AddProcessEnv(cbArg, cbArgVal)
	}

	for cbEnv, cbEnvVal := range c.envs {
		g.AddProcessEnv(cbEnv, cbEnvVal)
	}

	if c.stage.docker.Config.WorkingDir != "" {
		g.SetProcessCwd(c.stage.docker.Config.WorkingDir)
	}

	for _, mp := range constant.MaskedPaths {
		g.AddLinuxMaskedPaths(mp)
	}

	for _, rp := range constant.ReadonlyPaths {
		g.AddLinuxReadonlyPaths(rp)
	}

	// add capability
	for _, cap := range c.stage.builder.buildOpts.CapAddList {
		if aerr := g.AddProcessCapability(cap); aerr != nil {
			return nil, errors.Wrapf(aerr, "runner: add process capability %v failed", cap)
		}
	}

	return g.Config, nil
}

func setupBindFiles(bundlePath string) (map[string]string, error) {
	const bindFilesNum = 2
	bindFiles := make(map[string]string, bindFilesNum)

	hostsFile, err := generateHosts(bundlePath)
	if err != nil {
		return nil, err
	}
	bindFiles[constant.HostsFilePath] = hostsFile

	resolvFile, err := generateResolv(bundlePath)
	if err != nil {
		return nil, err
	}
	bindFiles[constant.ResolvFilePath] = resolvFile

	return bindFiles, nil
}

func generateHosts(bundlePath string) (string, error) {
	if err := util.CheckFileSize(constant.HostsFilePath, constant.MaxFileSize); err != nil {
		return "", err
	}

	hostsContent, err := ioutil.ReadFile(constant.HostsFilePath)
	if err != nil {
		return "", errors.Errorf("read %s err: %v", constant.HostsFilePath, err)
	}

	hostsFile, err := securejoin.SecureJoin(bundlePath, "hosts")
	if err != nil {
		return "", errors.Errorf("secureJoin bundlePath and hosts err: %v", err)
	}

	// DefaultSharedFileMode is necessary for hosts file when running container
	if err := ioutils.AtomicWriteFile(hostsFile, hostsContent, constant.DefaultSharedFileMode); err != nil {
		return "", errors.Errorf("write %s content to hosts file %s err: %v", constant.HostsFilePath, hostsFile, err)
	}

	return hostsFile, nil
}

func generateResolv(bundlePath string) (string, error) {
	if err := util.CheckFileSize(constant.ResolvFilePath, constant.MaxFileSize); err != nil {
		return "", err
	}

	resolvContent, err := ioutil.ReadFile(constant.ResolvFilePath)
	if err != nil {
		return "", errors.Errorf("read %s err: %v", constant.ResolvFilePath, err)
	}

	resolvFile, err := securejoin.SecureJoin(bundlePath, "resolv.conf")
	if err != nil {
		return "", errors.Errorf("secureJoin bundlePath and resolv.conf err: %v", err)
	}

	searchDomains := resolvconf.GetSearchDomains(resolvContent)
	nameServers := resolvconf.GetNameservers(resolvContent, types.IP)
	options := resolvconf.GetOptions(resolvContent)
	// DefaultSharedFileMode is necessary for resolv.conf file when running container
	if _, err = resolvconf.Build(resolvFile, nameServers, searchDomains, options); err != nil {
		return "", errors.Errorf("build resolv.conf err: %v", err)
	}

	return resolvFile, nil
}

func setupMounts(spec *specs.Spec, bindFiles map[string]string) {
	// setup sysfs cgroup mounts
	sysfsMounts := []specs.Mount{{
		Source:      "cgroup",
		Destination: "/sys/fs/cgroup",
		Type:        "cgroup",
		Options:     []string{"nosuid", "noexec", "nodev", "relatime", "ro"},
	}}

	// setup bind files mounts
	var bindFilesMounts []specs.Mount
	for dest, src := range bindFiles {
		bindFilesMounts = append(bindFilesMounts, specs.Mount{
			Source:      src,
			Destination: dest,
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		})
	}

	// add all mounts
	var mounts []specs.Mount
	alreadyMounts := make(map[string]bool, len(spec.Mounts)+len(sysfsMounts)+len(bindFilesMounts))
	for _, mount := range append(append(sysfsMounts, bindFilesMounts...), spec.Mounts...) {
		// if destination already mounts something, skip
		if _, ok := alreadyMounts[mount.Destination]; ok {
			continue
		}
		alreadyMounts[mount.Destination] = true
		mounts = append(mounts, mount)
	}

	spec.Mounts = mounts
}
