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
// Create: 2020-01-20
// Description: This file is used for isula-build daemon

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/containers/storage/pkg/reexec"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	"isula.org/isula-build/cmd/daemon/config"
	"isula.org/isula-build/daemon"
	_ "isula.org/isula-build/exporter/register"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/version"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const lockFileName = "isula-builder.lock"

var daemonOpts daemon.Options

func newDaemonCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "isula-builder",
		Short: "isula-build daemon for container image building",
		RunE:  runDaemon,
		Args:  util.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return before(cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s, build %s", version.Version, version.GitCommit),
	}
	rootCmd.PersistentFlags().BoolVarP(&daemonOpts.Debug, "debug", "D", false, "Open debug mode")
	rootCmd.PersistentFlags().BoolVarP(&daemonOpts.Experimental, "experimental", "", false, "Enable experimental features")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.DataRoot, "dataroot", constant.DefaultDataRoot, "Persistent dir")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.RunRoot, "runroot", constant.DefaultRunRoot, "Runtime dir")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.Group, "group", "isula", "User group for unix socket isula-build.sock")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.StorageDriver, "storage-driver", "overlay", "Storage-driver")
	rootCmd.PersistentFlags().StringSliceVar(&daemonOpts.StorageOpts, "storage-opt", []string{}, "Storage driver option")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.LogLevel, "log-level", "info", "Log level to be used. Either \"debug\", \"info\", \"warn\" or \"error\"")
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Version for isula-build daemon")

	rootCmd.SetFlagErrorFunc(util.FlagErrorFunc)
	addCommands(rootCmd)

	return rootCmd
}

func addCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		completionCmd,
	)
}

var completionCmd = &cobra.Command{
	Use:    "completion",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Root().GenBashCompletion(os.Stdout) // nolint
	},
}

func runDaemon(cmd *cobra.Command, args []string) error {
	store, err := store.GetStore()
	if err != nil {
		return err
	}
	// cleanup the residual container store if it exists
	store.CleanContainers()
	// Ensure we have only one daemon running at the same time
	lock, err := util.SetDaemonLock(daemonOpts.RunRoot, lockFileName)
	if err != nil {
		return err
	}
	defer func() {
		if uerr := lock.Unlock(); uerr != nil {
			logrus.Errorf("Unlock file %s failed: %v", lock.Path(), uerr)
		} else if rerr := os.RemoveAll(lock.Path()); rerr != nil {
			logrus.Errorf("Remove lock file %s failed: %v", lock.Path(), rerr)
		}
	}()

	d, err := daemon.NewDaemon(daemonOpts, &store)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := d.Cleanup(); cerr != nil {
			if err == nil {
				err = cerr
			} else {
				logrus.Errorf("Cleanup resources failed: %v", cerr)
			}
		}
	}()

	return d.Run()
}

func main() {
	if reexec.Init() {
		return
	}

	cmd := newDaemonCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(constant.DefaultFailedCode)
	}
}

func initLogging() error {
	logrusLvl, err := logrus.ParseLevel(daemonOpts.LogLevel)
	if err != nil {
		return errors.Wrapf(err, "unable to parse log level")
	}
	logrus.SetLevel(logrusLvl)
	if daemonOpts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return nil
}

// before parses input params for runDaemon()
func before(cmd *cobra.Command) error {
	if !util.SetUmask() {
		return errors.New("setting umask failed")
	}

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	if err := validateConfigFileAndMerge(cmd); err != nil {
		return err
	}
	if err := setStoreAccordingToDaemonOpts(); err != nil {
		return err
	}

	if err := initLogging(); err != nil {
		return err
	}

	if err := setupWorkingDirectories(); err != nil {
		return err
	}

	image.SetSystemContext(daemonOpts.DataRoot)

	return nil
}

func setStoreAccordingToDaemonOpts() error {
	runRoot, err := securejoin.SecureJoin(daemonOpts.RunRoot, "storage")
	if err != nil {
		return err
	}
	dataRoot, err := securejoin.SecureJoin(daemonOpts.DataRoot, "storage")
	if err != nil {
		return err
	}

	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		RunRoot:      runRoot,
		DataRoot:     dataRoot,
		Driver:       daemonOpts.StorageDriver,
		DriverOption: util.CopyStrings(daemonOpts.StorageOpts),
	})

	return nil
}

func loadConfig(path string) (config.TomlConfig, error) {
	var conf config.TomlConfig
	if err := util.CheckFileInfoAndSize(path, constant.MaxFileSize); err != nil {
		return conf, err
	}

	configData, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return conf, err
	}
	_, err = toml.Decode(string(configData), &conf)

	return conf, err
}

func mergeStorageConfig(cmd *cobra.Command) error {
	store.SetDefaultConfigFilePath(constant.StorageConfigPath)
	option, err := store.GetDefaultStoreOptions(true)
	if err != nil {
		return err
	}

	if !cmd.Flag("runroot").Changed && option.RunRoot != "" {
		daemonOpts.RunRoot = option.RunRoot
	}
	if !cmd.Flag("dataroot").Changed && option.GraphRoot != "" {
		daemonOpts.DataRoot = option.GraphRoot
	}
	if !cmd.Flag("storage-driver").Changed && option.GraphDriverName != "" {
		daemonOpts.StorageDriver = option.GraphDriverName
	}
	if !cmd.Flag("storage-opt").Changed && len(option.GraphDriverOptions) > 0 {
		daemonOpts.StorageOpts = option.GraphDriverOptions
	}

	return nil
}

func mergeConfig(conf config.TomlConfig, cmd *cobra.Command) error {
	if conf.Debug && !cmd.Flag("debug").Changed {
		daemonOpts.Debug = true
	}
	if conf.Experimental && !cmd.Flag("experimental").Changed {
		daemonOpts.Experimental = true
	}
	if conf.LogLevel != "" && !cmd.Flag("log-level").Changed {
		daemonOpts.LogLevel = conf.LogLevel
	}
	if conf.Group != "" && !cmd.Flag("group").Changed {
		daemonOpts.Group = conf.Group
	}
	if conf.Runtime != "" {
		daemonOpts.RuntimePath = conf.Runtime
	}
	if conf.RunRoot != "" && !cmd.Flag("runroot").Changed {
		daemonOpts.RunRoot = conf.RunRoot
	}
	if conf.DataRoot != "" && !cmd.Flag("dataroot").Changed {
		daemonOpts.DataRoot = conf.DataRoot
	}

	return nil
}

func setupWorkingDirectories() error {
	if daemonOpts.RunRoot == daemonOpts.DataRoot {
		return errors.Errorf("runroot(%q) and dataroot(%q) must be different paths", daemonOpts.RunRoot, daemonOpts.DataRoot)
	}

	buildTmpDir, err := securejoin.SecureJoin(daemonOpts.DataRoot, constant.DataRootTmpDirPrefix)
	if err != nil {
		return err
	}
	dirs := []string{daemonOpts.DataRoot, daemonOpts.RunRoot, buildTmpDir}
	for _, dir := range dirs {
		if !filepath.IsAbs(dir) {
			return errors.Errorf("%q not an absolute dir, the \"dataroot\" and \"runroot\" must be an absolute path", dir)
		}

		if exist, err := util.IsExist(dir); err != nil {
			return err
		} else if !exist {
			if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
				return errors.Wrapf(err, "create directory for %q failed", dir)
			}
			continue
		}
		if !util.IsDirectory(dir) {
			return errors.Errorf("%q is not a directory", dir)
		}
	}

	// change config root owner as group current defined
	if err := util.ChangeGroup(constant.ConfigRoot, daemonOpts.Group); err != nil {
		logrus.Errorf("Chown for %s failed: %v", constant.ConfigRoot, err)
		return err
	}

	return nil
}

func validateConfigFileAndMerge(cmd *cobra.Command) error {
	confFiles := []struct {
		path        string
		needed      bool
		mergeConfig func(cmd *cobra.Command) error
	}{
		{path: constant.StorageConfigPath, needed: false, mergeConfig: mergeStorageConfig},
		{path: constant.RegistryConfigPath, needed: false, mergeConfig: nil},
		// policy.json file must exists
		{path: constant.SignaturePolicyPath, needed: true, mergeConfig: nil},
		// main configuration comes last for the final merge operation
		{path: constant.ConfigurationPath, needed: false, mergeConfig: loadMainConfiguration},
	}

	for _, file := range confFiles {
		if exist, err := util.IsExist(file.path); !exist {
			if !file.needed {
				logrus.Warnf("Config file %q missing, the default configuration is used", file.path)
				continue
			}

			if err != nil {
				return errors.Wrapf(err, "check config file %q failed", file.path)
			}
			return errors.Errorf("config file %q is not exist", file.path)
		}

		if err := util.CheckFileInfoAndSize(file.path, constant.MaxFileSize); err != nil {
			return err
		}
		if file.mergeConfig == nil {
			continue
		}
		if err := file.mergeConfig(cmd); err != nil {
			return err
		}
	}

	return nil
}

func loadMainConfiguration(cmd *cobra.Command) error {
	conf, err := loadConfig(constant.ConfigurationPath)
	if err != nil {
		logrus.Errorf("Load and parse main config file failed: %v", err)
		os.Exit(constant.DefaultFailedCode)
	}

	if err = mergeConfig(conf, cmd); err != nil {
		return err
	}

	return nil
}
