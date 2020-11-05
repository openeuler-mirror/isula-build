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
// Create: 2020-07-16
// Description: This file is used for command import

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	maxTarballSize = 1024 * 1024 * 1024 // support tarball max size at most 1G
	importExample  = `isula-build ctr-img import busybox.tar busybox:isula`
	importArgsLen  = 1
)

type importOptions struct {
	source    string
	reference string
	importID  string
}

var importOpts importOptions

// NewImportCmd returns import command
func NewImportCmd() *cobra.Command {
	importCmd := &cobra.Command{
		Use:     "import FILE [REPOSITORY[:TAG]]",
		Short:   "Import the base image from a tarball to the image store",
		Example: importExample,
		RunE:    importCommand,
	}
	return importCmd
}

func importCommand(c *cobra.Command, args []string) error {
	if len(args) < importArgsLen {
		return errors.New("requires at least one argument")
	}
	if err := util.CheckFileSize(args[0], maxTarballSize); err != nil {
		return err
	}
	importOpts.source = args[0]
	if len(args) > importArgsLen {
		importOpts.reference = args[1]
	}

	ctx := context.TODO()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}
	return runImport(ctx, cli)
}

func runImport(ctx context.Context, cli Cli) error {
	if importOpts.reference != "" {
		if _, err := dockerref.Parse(importOpts.reference); err != nil {
			return err
		}
	}

	if !filepath.IsAbs(importOpts.source) {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.New("get current path failed")
		}
		importOpts.source = util.MakeAbsolute(importOpts.source, pwd)
	}

	importOpts.importID = stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]

	stream, err := cli.Client().Import(ctx, &pb.ImportRequest{
		Source:    importOpts.source,
		Reference: importOpts.reference,
		ImportID:  importOpts.importID,
	})
	if err != nil {
		return err
	}

	for {
		msg, rErr := stream.Recv()
		if msg != nil {
			fmt.Print(msg.Log)
		}
		if rErr != nil {
			if rErr == io.EOF {
				return nil
			}
			return rErr
		}
	}
}
