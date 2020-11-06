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
// Description: This file is used for "push" command

package main

import (
	"context"
	"fmt"
	"io"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
)

const (
	pushExample = `isula-build ctr-img push registry.example.com/repository:tag`
)

// NewPushCmd returns push command
func NewPushCmd() *cobra.Command {
	pushCmd := &cobra.Command{
		Use:     "push REPOSITORY[:TAG]",
		Short:   "Push image to remote repository",
		Example: pushExample,
		RunE:    pushCommand,
	}
	return pushCmd
}

func pushCommand(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("push requires exactly one argument")
	}
	if _, err := dockerref.Parse(args[0]); err != nil {
		return err
	}

	ctx := context.TODO()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runPush(ctx, cli, args[0])
}

func runPush(ctx context.Context, cli Cli, imageName string) error {
	pushID := stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]

	pushStream, err := cli.Client().Push(ctx, &pb.PushRequest{
		PushID:    pushID,
		ImageName: imageName,
	})
	if err != nil {
		return err
	}
	for {
		msg, rErr := pushStream.Recv()
		if msg != nil {
			fmt.Print(msg.Response)
		}

		if rErr != nil {
			if rErr == io.EOF {
				fmt.Printf("Push success with image: %s\n", imageName)
				return nil
			}
			return errors.Errorf("push image failed: %v", rErr)
		}
	}
}
