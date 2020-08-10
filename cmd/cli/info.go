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
// Create: 2020-08-03
// Description: This file is used for info command

package main

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	infoExample = `isula-build info`
)

type infoOptions struct {
	humanReadable bool
}

var infoOpts infoOptions

// NewInfoCmd returns info command
func NewInfoCmd() *cobra.Command {
	// infoCmd represents the "info" command
	infoCmd := &cobra.Command{
		Use:     "info [FLAGS]",
		Short:   "Show isula-build system information",
		RunE:    infoCommand,
		Args:    util.NoArgs,
		Example: infoExample,
	}

	infoCmd.PersistentFlags().BoolVarP(&infoOpts.humanReadable, "human-readable", "H", false,
		"print memory info in human readable format, use powers of 1000")

	return infoCmd
}

func infoCommand(c *cobra.Command, args []string) error {
	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	infoData, err := getInfo(ctx, cli)
	if err != nil {
		return err
	}

	printInfo(infoData)

	return nil
}

func getInfo(ctx context.Context, cli Cli) (*pb.InfoResponse, error) {
	return cli.Client().Info(ctx, &types.Empty{})
}

func printInfo(infoData *pb.InfoResponse) {
	fmt.Println("General:")
	if infoOpts.humanReadable {
		fmt.Println("  MemTotal:    ", util.FormatSize(float64(infoData.MemInfo.MemTotal)))
		fmt.Println("  MemFree:     ", util.FormatSize(float64(infoData.MemInfo.MemFree)))
		fmt.Println("  SwapTotal:   ", util.FormatSize(float64(infoData.MemInfo.SwapTotal)))
		fmt.Println("  SwapFree:    ", util.FormatSize(float64(infoData.MemInfo.SwapFree)))
	} else {
		fmt.Println("  MemTotal:    ", infoData.MemInfo.MemTotal)
		fmt.Println("  MemFree:     ", infoData.MemInfo.MemFree)
		fmt.Println("  SwapTotal:   ", infoData.MemInfo.SwapTotal)
		fmt.Println("  SwapFree:    ", infoData.MemInfo.SwapFree)
	}
	fmt.Println("  OCI Runtime: ", infoData.OCIRuntime)
	fmt.Println("  DataRoot:    ", infoData.DataRoot)
	fmt.Println("  RunRoot:     ", infoData.RunRoot)
	fmt.Println("  Builders:    ", infoData.BuilderNum)
	fmt.Println("  Goroutines:  ", infoData.GoRoutines)

	fmt.Println("Store:")
	fmt.Println("  Storage Driver:    ", infoData.StorageInfo.StorageDriver)
	fmt.Println("  Backing Filesystem:", infoData.StorageInfo.StorageBackingFs)

	fmt.Println("Registry:")
	fmt.Println("  Search Registries:")
	for _, registry := range infoData.RegistryInfo.RegistriesSearch {
		fmt.Println("   ", registry)
	}
	fmt.Println("  Insecure Registries:")
	for _, registry := range infoData.RegistryInfo.RegistriesInsecure {
		fmt.Println("   ", registry)
	}
}
