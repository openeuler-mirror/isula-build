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
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/util"
)

const (
	infoExample = `isula-build info
isula-build info -V -H`
	binaryPrefixBase = 1024
	formatBase       = 10
)

type infoOptions struct {
	humanReadable bool
	verbose       bool
}

type sysMemInfo struct {
	memTotal  string
	memFree   string
	swapTotal string
	swapFree  string
}

type runtimeMemInfo struct {
	memSys          string
	memHeapSys      string
	memHeapAlloc    string
	memHeapInUse    string
	memHeapIdle     string
	memHeapReleased string
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
		"Print memory info in human readable format, use powers of 1000")
	infoCmd.PersistentFlags().BoolVarP(&infoOpts.verbose, "verbose", "V", false,
		"Print runtime memory info")

	return infoCmd
}

func infoCommand(c *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("invalid args for info command")
	}
	if c.Flag("verbose").Changed {
		infoOpts.verbose = true
	}
	if c.Flag("human-readable").Changed {
		infoOpts.humanReadable = true
	}

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return runInfo(ctx, cli)
}

func runInfo(ctx context.Context, cli Cli) error {
	req := &pb.InfoRequest{
		Verbose: infoOpts.verbose,
	}
	resp, err := cli.Client().Info(ctx, req)
	if err != nil {
		return err
	}
	printInfo(resp)

	return nil
}

func printInfo(infoData *pb.InfoResponse) {
	var (
		sysMem     sysMemInfo
		runtimeMem runtimeMemInfo
	)

	if infoOpts.humanReadable {
		sysMem.memTotal = util.FormatSize(float64(infoData.MemInfo.MemTotal), binaryPrefixBase)
		sysMem.memFree = util.FormatSize(float64(infoData.MemInfo.MemFree), binaryPrefixBase)
		sysMem.swapTotal = util.FormatSize(float64(infoData.MemInfo.SwapTotal), binaryPrefixBase)
		sysMem.swapFree = util.FormatSize(float64(infoData.MemInfo.SwapFree), binaryPrefixBase)
		if infoOpts.verbose {
			runtimeMem.memSys = util.FormatSize(float64(infoData.MemStat.MemSys), binaryPrefixBase)
			runtimeMem.memHeapSys = util.FormatSize(float64(infoData.MemStat.HeapSys), binaryPrefixBase)
			runtimeMem.memHeapAlloc = util.FormatSize(float64(infoData.MemStat.HeapAlloc), binaryPrefixBase)
			runtimeMem.memHeapInUse = util.FormatSize(float64(infoData.MemStat.HeapInUse), binaryPrefixBase)
			runtimeMem.memHeapIdle = util.FormatSize(float64(infoData.MemStat.HeapIdle), binaryPrefixBase)
			runtimeMem.memHeapReleased = util.FormatSize(float64(infoData.MemStat.HeapReleased), binaryPrefixBase)
		}
	} else {
		sysMem.memTotal = strconv.FormatInt(infoData.MemInfo.MemTotal, formatBase)
		sysMem.memFree = strconv.FormatInt(infoData.MemInfo.MemFree, formatBase)
		sysMem.swapTotal = strconv.FormatInt(infoData.MemInfo.SwapTotal, formatBase)
		sysMem.swapFree = strconv.FormatInt(infoData.MemInfo.SwapFree, formatBase)
		if infoOpts.verbose {
			runtimeMem.memSys = strconv.FormatUint(infoData.MemStat.MemSys, formatBase)
			runtimeMem.memHeapSys = strconv.FormatUint(infoData.MemStat.HeapSys, formatBase)
			runtimeMem.memHeapAlloc = strconv.FormatUint(infoData.MemStat.HeapAlloc, formatBase)
			runtimeMem.memHeapInUse = strconv.FormatUint(infoData.MemStat.HeapInUse, formatBase)
			runtimeMem.memHeapIdle = strconv.FormatUint(infoData.MemStat.HeapIdle, formatBase)
			runtimeMem.memHeapReleased = strconv.FormatUint(infoData.MemStat.HeapReleased, formatBase)
		}
	}
	fmt.Println("General:")
	fmt.Println("  MemTotal:    ", sysMem.memTotal)
	fmt.Println("  MemFree:     ", sysMem.memFree)
	fmt.Println("  SwapTotal:   ", sysMem.swapTotal)
	fmt.Println("  SwapFree:    ", sysMem.swapFree)
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
	if infoOpts.verbose {
		fmt.Println("Runtime:")
		fmt.Println("  MemSys:          ", runtimeMem.memSys)
		fmt.Println("  HeapSys:         ", runtimeMem.memHeapSys)
		fmt.Println("  HeapAlloc:       ", runtimeMem.memHeapAlloc)
		fmt.Println("  MemHeapInUse:    ", runtimeMem.memHeapInUse)
		fmt.Println("  MemHeapIdle:     ", runtimeMem.memHeapIdle)
		fmt.Println("  MemHeapReleased: ", runtimeMem.memHeapReleased)
	}
}
