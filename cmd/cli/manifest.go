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
// Create: 2020-12-01
// Description: This file is used for manifest command.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
	_ "isula.org/isula-build/exporter/register"
)

const (
	manifestCreateExample = `isula-build manifest create openeuler
isula-build manifest create openeuler localhost:5000/openeuler_x86:latest`
	manifestAnnotateExample = `isula-build manifest annotate --os linux --arch arm64 openeuler localhost:5000/openeuler_aarch64:latest`
	manifestInspectExample  = `isula-build manifest inspect openeuler:latest`
	manifestPushExample     = `isula-build manifest push openeuler:latest localhost:5000/openeuler`
)

type annotateOptions struct {
	imageArch      string
	imageOS        string
	imageOSFeature []string
	imageVariant   string
}

var annotateOpts annotateOptions

// NewManifestCmd returns manifest operations commands
func NewManifestCmd() *cobra.Command {
	manifestCmd := &cobra.Command{
		Use:   "manifest",
		Short: "Manipulate manifest lists",
	}
	manifestCmd.AddCommand(
		NewManifestCreateCmd(),
		NewManifestAnnotateCmd(),
		NewManifestInspectCmd(),
		NewManifestPushCmd(),
	)

	return manifestCmd
}

// NewManifestCreateCmd returns manifest create command
func NewManifestCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:                   "create MANIFEST_LIST MANIFEST [MANIFEST...] ",
		Short:                 "Create a local manifest list",
		Example:               manifestCreateExample,
		RunE:                  manifestCreateCommand,
		DisableFlagsInUseLine: true,
	}

	return createCmd
}

// NewManifestAnnotateCmd returns manifest annotate command
func NewManifestAnnotateCmd() *cobra.Command {
	annotateCmd := &cobra.Command{
		Use:     "annotate [FLAGS] MANIFEST_LIST MANIFEST",
		Short:   "Annotate a local manifest list",
		Example: manifestAnnotateExample,
		RunE:    manifestAnnotateCommand,
	}

	annotateCmd.PersistentFlags().StringVar(&annotateOpts.imageArch, "arch", "", "Set architecture")
	annotateCmd.PersistentFlags().StringVar(&annotateOpts.imageOS, "os", "", "Set operating system")
	annotateCmd.PersistentFlags().StringSliceVar(&annotateOpts.imageOSFeature, "os-features", []string{}, "Set operating system feature")
	annotateCmd.PersistentFlags().StringVar(&annotateOpts.imageVariant, "variant", "", "Set architecture variant")

	return annotateCmd
}

// NewManifestInspectCmd returns manifest inspect command
func NewManifestInspectCmd() *cobra.Command {
	inspectCmd := &cobra.Command{
		Use:                   "inspect MANIFEST_LIST",
		Short:                 "Inspect a local manifest list",
		Example:               manifestInspectExample,
		RunE:                  manifestInspectCommand,
		DisableFlagsInUseLine: true,
	}

	return inspectCmd
}

// NewManifestPushCmd returns manifest push command
func NewManifestPushCmd() *cobra.Command {
	pushCmd := &cobra.Command{
		Use:                   "push MANIFEST_LIST DEST",
		Short:                 "Push a local manifest list to a repository",
		Example:               manifestPushExample,
		RunE:                  manifestPushCommand,
		DisableFlagsInUseLine: true,
	}

	return pushCmd
}

func manifestCreateCommand(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("please specify a name to manifest list")
	}

	listName := args[0]
	manifestsName := args[1:]

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runManifestCreate(ctx, cli, listName, manifestsName)
}

func runManifestCreate(ctx context.Context, cli Cli, listName string, manifestsName []string) error {
	resp, err := cli.Client().ManifestCreate(ctx, &pb.ManifestCreateRequest{
		ManifestList: listName,
		Manifests:    manifestsName,
	})
	if err != nil {
		return err
	}

	fmt.Println(resp.ImageID)

	return nil
}

func manifestAnnotateCommand(c *cobra.Command, args []string) error {
	var validArgsLength = 2
	if len(args) != validArgsLength {
		return errors.New("please specify the manifest list and the image name")
	}

	listName := args[0]
	manifestName := args[1]

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runManifestAnnotate(ctx, cli, listName, manifestName)
}

func runManifestAnnotate(ctx context.Context, cli Cli, listName, manifestName string) error {
	if _, err := cli.Client().ManifestAnnotate(ctx, &pb.ManifestAnnotateRequest{
		ManifestList: listName,
		Manifest:     manifestName,
		Arch:         annotateOpts.imageArch,
		Os:           annotateOpts.imageOS,
		OsFeatures:   annotateOpts.imageOSFeature,
		Variant:      annotateOpts.imageVariant,
	}); err != nil {
		return err
	}

	fmt.Println("manifest annotate succeed")

	return nil
}

func manifestInspectCommand(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("please specify the manifest list name")
	}

	if len(args) > 1 {
		return errors.New("only one manifest list can be specified")
	}

	listName := args[0]

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runManifestInspect(ctx, cli, listName)
}

func runManifestInspect(ctx context.Context, cli Cli, listName string) error {
	resp, err := cli.Client().ManifestInspect(ctx, &pb.ManifestInspectRequest{
		ManifestList: listName,
	})
	if err != nil {
		return err
	}

	var b bytes.Buffer
	if err = json.Indent(&b, resp.Data, "", "    "); err != nil {
		return errors.Wrap(err, "display manifest error")
	}

	fmt.Println(b.String())

	return nil
}

func manifestPushCommand(c *cobra.Command, args []string) error {
	if len(args) != 2 {
		return errors.New("please specify the manifest list name and destination repository")
	}

	listName := args[0]
	dest := args[1]

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runManifestPush(ctx, cli, listName, dest)
}

func runManifestPush(ctx context.Context, cli Cli, listName, dest string) error {
	resp, err := cli.Client().ManifestPush(ctx, &pb.ManifestPushRequest{
		ManifestList: listName,
		Dest:         dest,
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
			fmt.Print(msg.Result)
		}
	}

	fmt.Println("manifest list push succeed")

	return nil
}
