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
// Create: 2020-01-20
// Description: This file is used for remove command

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
)

type removeOptions struct {
	all   bool
	prune bool
}

var removeOpts removeOptions

const (
	removeExample = `isula-build ctr-img rm <imageID>
isula-build ctr-img rm --prune
isula-build ctr-img rm --all`
)

// NewRemoveCmd returns remove command
func NewRemoveCmd() *cobra.Command {
	// removeCmd represents the "rm" command
	removeCmd := &cobra.Command{
		Use:     "rm IMAGE [IMAGE...] [FLAGS]",
		Short:   "Remove one or more locally stored images",
		Example: removeExample,
		RunE:    removeCommand,
	}
	removeCmd.PersistentFlags().BoolVarP(&removeOpts.all, "all", "a", false, "Remove all images")
	removeCmd.PersistentFlags().BoolVarP(&removeOpts.prune, "prune", "p", false, "Remove all untagged images")
	return removeCmd
}

func removeCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runRemove(ctx, cli, args)
}

func runRemove(ctx context.Context, cli Cli, args []string) error {
	if err := checkArgsAndOptions(args); err != nil {
		return err
	}

	stream, err := cli.Client().Remove(ctx, &pb.RemoveRequest{
		ImageID: args,
		All:     removeOpts.all,
		Prune:   removeOpts.prune,
	})
	if err != nil {
		return err
	}

	for {
		msg, err := stream.Recv()
		if msg != nil {
			fmt.Println(msg.LayerMessage)
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func checkArgsAndOptions(args []string) error {
	if len(args) > 0 {
		if removeOpts.all {
			return errors.New("imageID is not allowed when using --all")
		}

		if removeOpts.prune {
			return errors.New("imageID is not allowed when using --prune")
		}
		return nil
	}

	if !removeOpts.all && !removeOpts.prune {
		return errors.New("imageID must be specified")
	}

	if removeOpts.all && removeOpts.prune {
		return errors.New("--prune is not allowed when using --all")
	}

	return nil
}
