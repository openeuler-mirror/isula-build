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
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type separatorSaveOption struct {
	baseImgName  string
	libImageName string
	renameFile   string
	destPath     string
	enabled      bool
}

type saveOptions struct {
	images []string
	sep    separatorSaveOption
	path   string
	saveID string
	format string
}

var saveOpts saveOptions

const (
	saveExample = `isula-build ctr-img save busybox:latest -o busybox.tar
isula-build ctr-img save 21c3e96ac411 -o myimage.tar
isula-build ctr-img save busybox:latest alpine:3.9 -o all.tar
isula-build ctr-img save app:latest -b busybox:latest -d Images
isula-build ctr-img save app:latest app1:latest -d Images -b busybox:latest -l lib:latest -r rename.json`
)

// NewSaveCmd cmd for container image saving
func NewSaveCmd() *cobra.Command {
	saveCmd := &cobra.Command{
		Use:     "save IMAGE [IMAGE...] FLAGS",
		Short:   "Save image to tarball",
		Example: saveExample,
		RunE:    saveCommand,
	}

	saveCmd.PersistentFlags().StringVarP(&saveOpts.path, "output", "o", "", "Path to save the tarball")
	saveCmd.PersistentFlags().StringVarP(&saveOpts.sep.destPath, "dest", "d", "", "Destination file directory to store separated images")
	saveCmd.PersistentFlags().StringVarP(&saveOpts.sep.baseImgName, "base", "b", "", "Base image name of separated images")
	saveCmd.PersistentFlags().StringVarP(&saveOpts.sep.libImageName, "lib", "l", "", "Lib image name of separated images")
	saveCmd.PersistentFlags().StringVarP(&saveOpts.sep.renameFile, "rename", "r", "", "Rename json file path of separated images")
	if util.CheckCliExperimentalEnabled() {
		saveCmd.PersistentFlags().StringVarP(&saveOpts.format, "format", "f", "oci", "Format of image saving to local tarball")
	} else {
		saveOpts.format = constant.DockerTransport
	}

	return saveCmd
}

func saveCommand(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := saveOpts.checkSaveOpts(args); err != nil {
		return err
	}

	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runSave(ctx, cli, args)
}

func (sep *separatorSaveOption) check(pwd string) error {
	if len(sep.baseImgName) == 0 {
		return errors.New("base image name(-b) must be provided")
	}
	if !util.IsValidImageName(sep.baseImgName) {
		return errors.Errorf("invalid base image name %s", sep.baseImgName)
	}
	if len(sep.libImageName) != 0 {
		if sep.libImageName == sep.baseImgName {
			return errors.New("base and lib images are the same")
		}
		if !util.IsValidImageName(sep.libImageName) {
			return errors.Errorf("invalid lib image name %s", sep.libImageName)
		}
	}
	if len(sep.destPath) == 0 {
		sep.destPath = "Images"
	}
	sep.destPath = util.MakeAbsolute(sep.destPath, pwd)
	if exist, err := util.IsExist(sep.destPath); err != nil {
		return errors.Wrap(err, "check dest path failed")
	} else if exist {
		return errors.Errorf("dest path already exist: %q, try to remove or rename it", sep.destPath)
	}
	if len(sep.renameFile) != 0 {
		sep.renameFile = util.MakeAbsolute(sep.renameFile, pwd)
	}

	return nil
}

func (opt *saveOptions) checkSaveOpts(args []string) error {
	if len(args) == 0 {
		return errors.New("save accepts at least one image")
	}

	if strings.Contains(opt.path, ":") || strings.Contains(opt.sep.destPath, ":") {
		return errors.Errorf("colon in path %q is not supported", opt.path)
	}
	pwd, err := os.Getwd()
	if err != nil {
		return errors.New("get current path failed")
	}

	// separator save
	if opt.sep.isEnabled() {
		if len(opt.path) != 0 {
			return errors.New("conflict flags between -o and [-b -l -r -d]")
		}
		// separate image only support docker image spec
		opt.format = constant.DockerTransport
		if err := opt.sep.check(pwd); err != nil {
			return err
		}
		opt.sep.enabled = true

		return nil
	}

	// normal save
	// only check oci format when doing normal save operation
	if len(opt.path) == 0 {
		return errors.New("output path(-o) should not be empty")
	}
	if opt.format == constant.OCITransport && len(args) >= 2 {
		return errors.New("oci image format now only supports saving single image")
	}
	if err := util.CheckImageFormat(opt.format); err != nil {
		return err
	}
	opt.path = util.MakeAbsolute(opt.path, pwd)
	if exist, err := util.IsExist(opt.path); err != nil {
		return errors.Wrap(err, "check output path failed")
	} else if exist {
		return errors.Errorf("output file already exist: %q, try to remove existing tarball or rename output file", opt.path)
	}
	return nil
}

func runSave(ctx context.Context, cli Cli, args []string) error {
	saveOpts.saveID = util.GenerateNonCryptoID()[:constant.DefaultIDLen]
	saveOpts.images = args

	sep := &pb.SeparatorSave{
		Base:    saveOpts.sep.baseImgName,
		Lib:     saveOpts.sep.libImageName,
		Rename:  saveOpts.sep.renameFile,
		Dest:    saveOpts.sep.destPath,
		Enabled: saveOpts.sep.enabled,
	}

	saveStream, err := cli.Client().Save(ctx, &pb.SaveRequest{
		Images: saveOpts.images,
		Path:   saveOpts.path,
		SaveID: saveOpts.saveID,
		Format: saveOpts.format,
		Sep:    sep,
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
			return errors.Errorf("save image failed: %v", err.Error())
		}
	}
}

func (sep *separatorSaveOption) isEnabled() bool {
	return util.AnyFlagSet(sep.baseImgName, sep.libImageName, sep.renameFile, sep.destPath)
}
