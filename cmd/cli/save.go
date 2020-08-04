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
// Create: 2020-07-31
// Description: This file is used for "save" command

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/util"
)

type saveOptions struct {
	image  string
	path   string
	saveID string
}

var saveOpts saveOptions

const (
	saveExample = `isula-build ctr-img save busybox:latest -o busybox.tar
isula-build ctr-img save 21c3e96ac411 -o myimage.tar`
)

// NewSaveCmd cmd for container image saving
func NewSaveCmd() *cobra.Command {
	saveCmd := &cobra.Command{
		Use:     "save",
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
	if len(args) != 1 {
		return errors.New("save accepts only one image")
	}

	if len(saveOpts.path) == 0 {
		return errors.New("output path should not be empty")
	}

	saveOpts.saveID = stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]

	if !filepath.IsAbs(saveOpts.path) {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.New("get current path failed")
		}
		saveOpts.path = util.MakeAbsolute(saveOpts.path, pwd)
	}

	saveOpts.image = args[0]

	saveStream, err := cli.Client().Save(ctx, &pb.SaveRequest{
		Image:  saveOpts.image,
		Path:   saveOpts.path,
		SaveID: saveOpts.saveID,
	})
	if err != nil {
		return err
	}

	fileChan := make(chan []byte, constant.BufferSize)
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(fileChan)
		for {
			msg, err := saveStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			fileChan <- msg.Data
			fmt.Print(msg.Log)
		}
		return nil
	})

	eg.Go(func() error {
		if err := exporter.ArchiveRecv(ctx, saveOpts.path, false, fileChan); err != nil {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		if rErr := os.Remove(saveOpts.path); rErr != nil {
			logrus.Warnf("Removing save output tarball %q failed: %v", saveOpts.path, rErr)
		}
		return errors.Errorf("save image failed: %v", err)
	}

	fmt.Printf("Save success with image: %s\n", saveOpts.image)
	return nil
}
