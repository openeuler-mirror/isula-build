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
	"os"

	"github.com/containers/storage/pkg/reexec"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	"isula.org/isula-build/daemon"
	_ "isula.org/isula-build/exporter/register"
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
