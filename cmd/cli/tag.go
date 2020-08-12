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
// Create: 2020-07-20
// Description: This file is used for tag command

package main

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
)

const (
	tagExample = `isula-build ctr-img tag a24bb4013296 busybox:latest
isula-build ctr-img tag busybox:v1.0 busybox:latest`
)

// NewTagCmd returns tag command
func NewTagCmd() *cobra.Command {
	// tagCmd represents the "tag" command
	tagCmd := &cobra.Command{
		Use:     "tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		Short:   "create a tag for source image",
		RunE:    tagCommand,
		Example: tagExample,
	}
	return tagCmd
}

func tagCommand(cmd *cobra.Command, args []string) error {
	const validTagArgsLen = 2
	if len(args) != validTagArgsLen {
		return errors.New("invalid args for tag command")
	}

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runTag(ctx, cli, args)
}

func runTag(ctx context.Context, cli Cli, args []string) error {
	_, err := cli.Client().Tag(ctx, &pb.TagRequest{
		Image: args[0],
		Tag:   args[1],
	})

	return err
}
