// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2020-10-15
// Description: This file is used for "pull" command

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	pullExample = `isula-build ctr-img pull registry.example.com/repository:tag`
)

// NewPullCmd returns pull command
func NewPullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:     "pull REPOSITORY[:TAG]",
		Short:   "Pull image from remote repository",
		Example: pullExample,
		RunE:    pullCommand,
	}
	return pullCmd
}

func pullCommand(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("pull requires exactly one argument")
	}

	ctx := context.TODO()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runPull(ctx, cli, args[0])
}

func runPull(ctx context.Context, cli Cli, imageName string) error {
	pullID := util.GenerateNonCryptoID()[:constant.DefaultIDLen]

	pullStream, err := cli.Client().Pull(ctx, &pb.PullRequest{
		PullID:    pullID,
		ImageName: imageName,
	})
	if err != nil {
		return err
	}
	for {
		msg, rErr := pullStream.Recv()
		if msg != nil {
			fmt.Print(msg.Response)
		}

		if rErr != nil {
			if rErr == io.EOF {
				fmt.Printf("Pull success with image: %s\n", imageName)
				return nil
			}
			return errors.Errorf("pull image failed: %v", rErr)
		}
	}
}
