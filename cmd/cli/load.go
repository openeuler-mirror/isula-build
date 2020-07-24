/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
 * isula-build licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: Feiyu Yang
 * Create: 2020-07-17
 * Description: This file is used for image load command
******************************************************************************/

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type loadOptions struct {
	path string
}

var loadOpts loadOptions

const (
	loadExample = `isula-build ctr-img load -i busybox.tar`
)

// NewLoadCmd returns image load command
func NewLoadCmd() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:     "load",
		Short:   "Load images",
		Example: loadExample,
		RunE:    loadCommand,
	}

	loadCmd.PersistentFlags().
		StringVarP(&loadOpts.path, "input", "i", "", "Path to local tarball")
	return loadCmd
}

func loadCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runLoad(ctx, cli, args)
}

func runLoad(ctx context.Context, cli Cli, args []string) error {
	if len(args) > 0 {
		return errors.New("load accepts no arguments")
	}

	// check input
	if len(loadOpts.path) == 0 {
		return errors.New("tarball path should not be empty")
	}

	if !filepath.IsAbs(loadOpts.path) {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.New("get current path failed")
		}
		loadOpts.path = util.MakeAbsolute(loadOpts.path, pwd)
	}

	if err := util.CheckLoadFile(loadOpts.path); err != nil {
		return err
	}

	resp, err := cli.Client().Load(ctx, &pb.LoadRequest{
		Path: loadOpts.path,
	})
	if err != nil {
		return err
	}

	if resp != nil {
		fmt.Printf("Imported image as %v\n", resp.ImageID)
	}

	return nil
}
