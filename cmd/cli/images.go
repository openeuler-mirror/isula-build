// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu
// Create: 2020-01-20
// Description: This file is used for command images

package main

import (
	"context"
	"fmt"

	"github.com/bndr/gotabulate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
)

const (
	// when list is empty, only print this head
	emptyStr = `-----------   ----   ---------   --------
REPOSITORY    TAG    IMAGE ID    CREATED
-----------   ----   ---------   --------`
)

const (
	imagesExample = `isula-build ctr-img images
isula-build ctr-img images <image name>`
)

// NewImagesCmd returns images command
func NewImagesCmd() *cobra.Command {
	// imagesCmd represents the "images" command
	imagesCmd := &cobra.Command{
		Use:     "images [REPOSITORY[:TAG]]",
		Short:   "List locally stored images",
		Example: imagesExample,
		RunE:    imagesCommand,
	}

	return imagesCmd
}

func imagesCommand(c *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("isula-build images requires at most one argument")
	}

	var image string
	if len(args) == 0 {
		image = ""
	} else {
		image = args[0]
	}

	ctx := context.TODO()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runList(ctx, cli, image)
}

func runList(ctx context.Context, cli Cli, image string) error {
	resp, err := cli.Client().List(ctx, &pb.ListRequest{
		ImageName: image,
	})
	if err != nil {
		return err
	}
	formatAndPrint(resp.Images)

	return nil
}

func formatAndPrint(images []*pb.ListResponse_ImageInfo) {
	lines := make([][]string, 0, len(images))
	title := []string{"REPOSITORY", "TAG", "IMAGE ID", "CREATED", "SIZE"}
	for _, image := range images {
		if image == nil {
			continue
		}
		line := []string{image.Repository, image.Tag, image.Id[:constant.DefaultIDLen], image.Created, image.Size_}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		fmt.Println(emptyStr)
		return
	}
	tabulate := gotabulate.Create(lines)
	tabulate.SetHeaders(title)
	tabulate.SetAlign("left")
	fmt.Print(tabulate.Render("simple"))
}
