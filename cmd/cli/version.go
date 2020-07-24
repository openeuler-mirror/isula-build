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
// Create: 2020-01-20
// Description: This file is used for version command

package main

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	"isula.org/isula-build/pkg/version"
	"isula.org/isula-build/util"
)

const (
	versionExample = `isula-build version`
)

// NewVersionCmd returns version command
func NewVersionCmd() *cobra.Command {
	// versionCmd represents the "version" command
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Show the isula-build version information",
		RunE:    versionCommand,
		Args:    util.NoArgs,
		Example: versionExample,
	}
	return versionCmd
}

func versionCommand(c *cobra.Command, args []string) error {
	var err error
	buildTime := int64(0)
	const base, baseSize = 10, 64
	if version.BuildInfo != "" {
		buildTime, err = strconv.ParseInt(version.BuildInfo, base, baseSize)
		if err != nil {
			return err
		}
	}

	// print out the client version information
	fmt.Println("Client:")
	fmt.Println("  Version:      ", version.Version)
	fmt.Println("  Go Version:   ", runtime.Version())
	fmt.Println("  Git Commit:   ", version.GitCommit)
	fmt.Println("  Built:        ", time.Unix(buildTime, 0).Format(time.ANSIC))
	fmt.Println("  OS/Arch:      ", runtime.GOOS+"/"+runtime.GOARCH)

	ctx := context.Background()
	cli, err := NewClient(ctx)
	if err != nil {
		return err
	}

	return getDaemonVersion(ctx, cli)
}

func getDaemonVersion(ctx context.Context, cli Cli) error {
	serverVersion, err := cli.Client().Version(ctx, &types.Empty{})
	if err != nil {
		return err
	}

	// print out the server version information
	fmt.Println("\nServer:")
	fmt.Println("  Version:      ", serverVersion.Version)
	fmt.Println("  Go Version:   ", serverVersion.GoVersion)
	fmt.Println("  Git Commit:   ", serverVersion.GitCommit)
	fmt.Println("  Built:        ", serverVersion.BuildTime)
	fmt.Println("  OS/Arch:      ", serverVersion.OsArch)

	return nil
}
