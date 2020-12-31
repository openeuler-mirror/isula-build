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
// Create: 2020-07-31
// Description: This file is used for "save" command

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type saveOptions struct {
	images []string
	path   string
	saveID string
}

var saveOpts saveOptions

const (
	saveExample = `isula-build ctr-img save busybox:latest -o busybox.tar
isula-build ctr-img save 21c3e96ac411 -o myimage.tar
isula-build ctr-img save busybox:latest alpine:3.9 -o all.tar`
)

// NewSaveCmd cmd for container image saving
func NewSaveCmd() *cobra.Command {
	saveCmd := &cobra.Command{
		Use:     "save IMAGE [IMAGE...] [FLAGS]",
		Short:   "Save image to tarball",
		Example: saveExample,
		RunE:    saveCommand,
	}

	saveCmd.PersistentFlags().StringVarP(&saveOpts.path, "output", "o", "", "Path to save the tarball")

	return saveCmd
}

func saveCommand(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runSave(ctx, cli, args)
}

func runSave(ctx context.Context, cli Cli, args []string) error {
	if len(args) == 0 {
		return errors.New("save accepts at least one image")
	}

	if len(saveOpts.path) == 0 {
		return errors.New("output path should not be empty")
	}

	saveOpts.saveID = stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]
	saveOpts.images = args

	if strings.Contains(saveOpts.path, ":") {
		return errors.Errorf("colon in path %q is not supported", saveOpts.path)
	}

	if !filepath.IsAbs(saveOpts.path) {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.New("get current path failed")
		}
		saveOpts.path = util.MakeAbsolute(saveOpts.path, pwd)
	}

	if util.IsExist(saveOpts.path) {
		return errors.Errorf("output file already exist: %q, try to remove existing tarball or rename output file", saveOpts.path)
	}

	saveStream, err := cli.Client().Save(ctx, &pb.SaveRequest{
		Images: saveOpts.images,
		Path:   saveOpts.path,
		SaveID: saveOpts.saveID,
	})
	if err != nil {
		return err
	}

	for {
		msg, err := saveStream.Recv()
		if msg != nil {
			fmt.Print(msg.Log)
		}

		if err != nil {
			if err == io.EOF {
				fmt.Printf("Save success with image: %s\n", saveOpts.images)
				return nil
			}
			return errors.Errorf("save image failed: %v", err)
		}
	}
}
