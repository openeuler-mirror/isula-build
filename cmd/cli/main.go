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
// Description: This file is isula-build client

package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	"isula.org/isula-build/pkg/version"
	"isula.org/isula-build/util"
)

type cliOptions struct {
	Debug    bool
	LogLevel string
	Timeout  string
}

var cliOpts cliOptions

func newCliCommand() *cobra.Command {
	// rootCmd represents the base command when called without any sub commands
	rootCmd := &cobra.Command{
		Use:   "isula-build",
		Short: "isula-build client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return before(cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s, build %s", version.Version, version.GitCommit),
	}
	setupRootCmd(rootCmd)
	addCommands(rootCmd)
	return rootCmd
}

func initLogging() error {
	logrusLvl, err := logrus.ParseLevel(cliOpts.LogLevel)
	if err != nil {
		return errors.Wrapf(err, "unable to parse log level")
	}
	logrus.SetLevel(logrusLvl)
	if cliOpts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetOutput(os.Stdout)
	return nil
}

func before(cmd *cobra.Command) error {
	switch cmd.Use {
	case "", "help", "version":
		return nil
	}

	if err := initLogging(); err != nil {
		return err
	}

	return nil
}

func main() {
	cmd := newCliCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(constant.DefaultFailedCode)
	}
}

func setupRootCmd(rootCmd *cobra.Command) {
	rootCmd.SetFlagErrorFunc(util.FlagErrorFunc)
	rootCmd.PersistentFlags().StringVar(&cliOpts.LogLevel, "log-level", "error", "Log level to be used. Either \"debug\", \"info\", \"warn\" or \"error\"")
	rootCmd.PersistentFlags().BoolVarP(&cliOpts.Debug, "debug", "D", false, "Open debug mode")
	rootCmd.PersistentFlags().StringVarP(&cliOpts.Timeout, "timeout", "t", "500ms", "Timeout for connecting to daemon")
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Version for isula-build client")
}

func addCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		NewContainerImageBuildCmd(),
		NewVersionCmd(),
		NewLoginCmd(),
		NewLogoutCmd(),
		NewInfoCmd(),
	)
}
