// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// iSula-Kits licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-06-04
// Description: This file is used for "logout" command

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	logoutExample = `isula-build logout mydockerhub.io
isula-build logout -a`
	logoutFailed = "\nLogout Failed\n"
)

type logoutOptions struct {
	server string
	all    bool
}

var logoutOpts logoutOptions

// NewLogoutCmd returns logout command
func NewLogoutCmd() *cobra.Command {
	// logoutCmd represents the "logout" command
	logoutCmd := &cobra.Command{
		Use:     "logout [SERVER] [FLAGS]",
		Short:   "Logout from an image registry",
		Example: logoutExample,
		RunE:    logoutCommand,
	}

	logoutCmd.PersistentFlags().BoolVarP(&logoutOpts.all, "all", "a", false, "Logout all registries")
	return logoutCmd
}

func logoutCommand(c *cobra.Command, args []string) error {
	if err := newLogoutOptions(c, args); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}
	msg, err := runLogout(ctx, cli)
	fmt.Println(msg)
	if err != nil {
		return err
	}
	return nil
}

func runLogout(ctx context.Context, cli Cli) (string, error) {
	req := &pb.LogoutRequest{
		Server: logoutOpts.server,
		All:    logoutOpts.all,
	}
	resp, err := cli.Client().Logout(ctx, req)
	if err != nil {
		return logoutFailed, err
	}
	return resp.Result, err
}

func newLogoutOptions(c *cobra.Command, args []string) error {
	// args can not more than one
	if len(args) > 1 {
		return errTooManyArgs
	}

	// no need args check when all flag is set
	if c.Flag("all").Changed {
		logoutOpts.all = true
		logoutOpts.server = ""
		return nil
	}

	// logout from single registry
	if len(args) == 0 {
		return errEmptyRegistry
	}

	// get registry address
	server, err := util.ParseServer(args[0])
	if err != nil {
		return err
	}
	logoutOpts.server = server

	return nil
}
