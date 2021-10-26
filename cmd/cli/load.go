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
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

type separatorLoadOption struct {
	app       string
	base      string
	lib       string
	dir       string
	skipCheck bool
	enabled   bool
}

type loadOptions struct {
	path   string
	loadID string
	sep    separatorLoadOption
}

var loadOpts loadOptions

const (
	loadExample = `isula-build ctr-img load -i busybox.tar
isula-build ctr-img load -i app:latest -d /home/Images
isula-build ctr-img load -i app:latest -d /home/Images -b /home/Images/base.tar.gz -l /home/Images/lib.tar.gz`
)

// NewLoadCmd returns image load command
func NewLoadCmd() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:     "load [FLAGS]",
		Short:   "Load images",
		Example: loadExample,
		Args:    util.NoArgs,
		RunE:    loadCommand,
	}

	loadCmd.PersistentFlags().StringVarP(&loadOpts.path, "input", "i", "", "Path to local tarball(or app image name when load separated images)")
	loadCmd.PersistentFlags().StringVarP(&loadOpts.sep.dir, "dir", "d", "", "Path to separated image tarballs directory")
	loadCmd.PersistentFlags().StringVarP(&loadOpts.sep.base, "base", "b", "", "Base image tarball path of separated images")
	loadCmd.PersistentFlags().StringVarP(&loadOpts.sep.lib, "lib", "l", "", "Library image tarball path of separated images")
	loadCmd.PersistentFlags().BoolVarP(&loadOpts.sep.skipCheck, "no-check", "", false, "Skip sha256 check sum for legacy separated images loading")

	return loadCmd
}

func loadCommand(cmd *cobra.Command, args []string) error {
	if err := loadOpts.checkLoadOpts(); err != nil {
		return errors.Wrapf(err, "check load options failed")
	}

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runLoad(ctx, cli)
}

func runLoad(ctx context.Context, cli Cli) error {
	loadOpts.loadID = util.GenerateNonCryptoID()[:constant.DefaultIDLen]
	sep := &pb.SeparatorLoad{
		App:       loadOpts.sep.app,
		Dir:       loadOpts.sep.dir,
		Base:      loadOpts.sep.base,
		Lib:       loadOpts.sep.lib,
		SkipCheck: loadOpts.sep.skipCheck,
		Enabled:   loadOpts.sep.enabled,
	}

	resp, err := cli.Client().Load(ctx, &pb.LoadRequest{
		Path:   loadOpts.path,
		LoadID: loadOpts.loadID,
		Sep:    sep,
	})
	if err != nil {
		return err
	}

	for {
		msg, rerr := resp.Recv()
		if rerr != nil {
			if rerr != io.EOF {
				return rerr
			}
			break
		}
		if msg != nil {
			fmt.Print(msg.Log)
		}
	}

	return err
}

func resolveLoadPath(path string) (string, error) {
	// check input
	if path == "" {
		return "", errors.New("tarball path should not be empty")
	}

	if !filepath.IsAbs(path) {
		pwd, err := os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, "get current path failed while loading image")
		}
		path = util.MakeAbsolute(path, pwd)
	}

	if err := util.CheckLoadFile(path); err != nil {
		return "", err
	}

	return path, nil
}

func (opt *loadOptions) checkLoadOpts() error {
	// normal load
	if !opt.sep.isEnabled() {
		path, err := resolveLoadPath(opt.path)
		if err != nil {
			return err
		}
		opt.path = path

		return nil
	}

	// load separated image
	opt.sep.enabled = true
	if len(opt.path) == 0 {
		return errors.New("app image should not be empty")
	}

	// Use opt.path as app image name when operating separated images
	// this can be mark as a switch for handling separated images
	opt.sep.app = opt.path

	if err := opt.sep.check(); err != nil {
		return err
	}

	return nil
}

func (sep *separatorLoadOption) isEnabled() bool {
	return util.AnyFlagSet(sep.dir, sep.base, sep.lib, sep.app)
}

func (sep *separatorLoadOption) check() error {
	pwd, err := os.Getwd()
	if err != nil {
		return errors.New("get current path failed")
	}
	if !util.IsValidImageName(sep.app) {
		return errors.Errorf("invalid image name: %s", sep.app)
	}

	if len(sep.base) != 0 {
		path, err := resolveLoadPath(sep.base)
		if err != nil {
			return errors.Wrap(err, "resolve base tarball path failed")
		}
		sep.base = path
	}
	if len(sep.lib) != 0 {
		path, err := resolveLoadPath(sep.lib)
		if err != nil {
			return errors.Wrap(err, "resolve lib tarball path failed")
		}
		sep.lib = path
	}
	if len(sep.dir) == 0 {
		return errors.New("image tarball directory should not be empty")
	}
	if !filepath.IsAbs(sep.dir) {
		sep.dir = util.MakeAbsolute(sep.dir, pwd)
	}
	if !util.IsExist(sep.dir) {
		return errors.Errorf("image tarball directory %s is not exist", sep.dir)
	}

	return nil
}
