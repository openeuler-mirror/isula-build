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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	file, err := os.Open(importOpts.source)
	if err != nil {
		return err
	}
	defer func() {
		if ferr := file.Close(); ferr != nil {
			logrus.Warnf("Close file %s failed", file.Name())
		}
	}()

	rpcCli, err := cli.Client().Import(ctx)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	buf := make([]byte, constant.BufferSize, constant.BufferSize)
	var length int
	for {
		length, err = reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err = rpcCli.Send(&pb.ImportRequest{
			Data:      buf[0:length],
			Reference: importOpts.reference,
		}); err != nil {
			return err
		}
	}

	resp, err := rpcCli.CloseAndRecv()
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("import failed, got nil response")
	}
	fmt.Printf("Import success with image id: %s\n", resp.ImageID)
	return nil
}
