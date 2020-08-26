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
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/containers/storage/pkg/reexec"
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
	rootCmd.PersistentFlags().StringVar(&daemonOpts.DataRoot, "dataroot", constant.DefaultDataRoot, "Persistent dir")
	rootCmd.PersistentFlags().StringVar(&daemonOpts.RunRoot, "runroot", constant.DefaultRunRoot, "Runtime dir")
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
	store.CleanContainerStore()
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

	d, err := daemon.NewDaemon(daemonOpts, store)
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

	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		RunRoot:      filepath.Join(daemonOpts.RunRoot, "storage"),
		DataRoot:     filepath.Join(daemonOpts.DataRoot, "storage"),
		Driver:       daemonOpts.StorageDriver,
		DriverOption: util.CopyStrings(daemonOpts.StorageOpts),
	})

	if err := checkAndValidateConfig(cmd); err != nil {
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

func loadConfig(path string) (config.TomlConfig, error) {
	var conf config.TomlConfig
	fi, err := os.Stat(path)
	if err != nil {
		return conf, err
	}

	if !fi.Mode().IsRegular() {
		return conf, errors.New("config file must be a regular file")
	}

	if err = util.CheckFileSize(path, constant.MaxFileSize); err != nil {
		return conf, err
	}
	configData, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return conf, err
	}
	_, err = toml.Decode(string(configData), &conf)

	return conf, err
}

func mergeStorageConfig(cmd *cobra.Command) {
	store.SetDefaultConfigFilePath(constant.StorageConfigPath)
	option, err := store.GetDefaultStoreOptions(true)
	if err == nil {
		if option.GraphDriverName != "" && !cmd.Flag("storage-driver").Changed {
			daemonOpts.StorageDriver = option.GraphDriverName
		}
		if len(option.GraphDriverOptions) > 0 && !cmd.Flag("storage-opt").Changed {
			daemonOpts.StorageOpts = option.GraphDriverOptions
		}
	}

	var storeOpt store.DaemonStoreOptions
	if option.RunRoot == "" {
		storeOpt.RunRoot = filepath.Join(daemonOpts.RunRoot, "storage")
	}
	if option.GraphRoot == "" {
		storeOpt.DataRoot = filepath.Join(daemonOpts.DataRoot, "storage")
	}
	if daemonOpts.StorageDriver != "" {
		storeOpt.Driver = daemonOpts.StorageDriver
	}
	if len(daemonOpts.StorageOpts) > 0 {
		storeOpt.DriverOption = util.CopyStrings(daemonOpts.StorageOpts)
	}
	store.SetDefaultStoreOptions(storeOpt)
}

func mergeConfig(conf config.TomlConfig, cmd *cobra.Command) {
	if strconv.FormatBool(conf.Debug) == "true" && !cmd.Flag("debug").Changed {
		daemonOpts.Debug = true
	}

	if conf.LogLevel != "" && !cmd.Flag("log-level").Changed {
		daemonOpts.LogLevel = conf.LogLevel
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
}

func setupWorkingDirectories() error {
	if daemonOpts.RunRoot == daemonOpts.DataRoot {
		return errors.Errorf("runroot(%q) and dataroot(%q) must be different paths", daemonOpts.RunRoot, daemonOpts.DataRoot)
	}

	dirs := []string{daemonOpts.DataRoot, daemonOpts.RunRoot}
	for _, dir := range dirs {
		if !filepath.IsAbs(dir) {
			return errors.Errorf("%q not an absolute dir, the \"dataroot\" and \"runroot\" must be an absolute path", dir)
		}

		if !util.IsExist(dir) {
			if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
				return errors.Wrapf(err, "create directory for %q failed", dir)
			}
			continue
		}
		if !util.IsDirectory(dir) {
			return errors.Errorf("%q is not a directory", dir)
		}
	}

	return nil
}

func checkAndValidateConfig(cmd *cobra.Command) error {
	// check if configuration.toml file exists, merge config if exists
	if !util.IsExist(constant.ConfigurationPath) {
		logrus.Warnf("Main config file missing, the default configuration is used")
	} else {
		conf, err := loadConfig(constant.ConfigurationPath)
		if err != nil {
			logrus.Errorf("Load and parse main config file failed: %v", err)
			os.Exit(constant.DefaultFailedCode)
		}

		mergeConfig(conf, cmd)
	}

	// file policy.json must be exist
	if !util.IsExist(constant.SignaturePolicyPath) {
		return errors.Errorf("policy config file %v is not exist", constant.SignaturePolicyPath)
	}

	// check all config files
	confFiles := []string{constant.RegistryConfigPath, constant.SignaturePolicyPath, constant.StorageConfigPath}
	for _, file := range confFiles {
		if util.IsExist(file) {
			fi, err := os.Stat(file)
			if err != nil {
				return errors.Wrapf(err, "stat file %q failed", file)
			}

			if !fi.Mode().IsRegular() {
				return errors.Errorf("file %s should be a regular file", fi.Name())
			}

			if err := util.CheckFileSize(file, constant.MaxFileSize); err != nil {
				return err
			}
		}
	}

	// if storage config file exists, merge storage config
	if util.IsExist(constant.StorageConfigPath) {
		mergeStorageConfig(cmd)
	}

	return nil
}
