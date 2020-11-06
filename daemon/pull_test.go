// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Weizheng Xing
// Create: 2020-11-02
// Description: This file tests Pull interface

package daemon

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"gotest.tools/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/store"
)

type daemonTestOptions struct {
	RootDir string
	Daemon  *Daemon
}

type controlPullServer struct {
	grpc.ServerStream
}

func (c *controlPullServer) Context() context.Context {
	return context.Background()
}

func (c *controlPullServer) Send(response *pb.PullResponse) error {
	return nil
}

func init() {
	reexec.Init()

}

func prepare(t *testing.T) daemonTestOptions {
	dOpt := daemonTestOptions{
		RootDir: "",
		Daemon:  nil,
	}
	dOpt.RootDir = fs.NewDir(t, t.Name()).Path()

	opt := &Options{
		DataRoot: dOpt.RootDir + "/data",
		RunRoot:  dOpt.RootDir + "/run",
	}
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: dOpt.RootDir + "/data",
		RunRoot:  dOpt.RootDir + "/run",
	})
	localStore, _ := store.GetStore()
	dOpt.Daemon = &Daemon{
		opts:       opt,
		localStore: &localStore,
	}
	dOpt.Daemon.NewBackend()
	return dOpt
}

func tmpClean(options daemonTestOptions) {
	if err := unix.Unmount(options.RootDir+"/data/overlay", 0); err != nil {
		fmt.Printf("umount dir %s failed: %v\n", options.RootDir+"/overlay", err)
	}

	if err := os.RemoveAll(options.RootDir); err != nil {
		fmt.Printf("remove test root dir %s failed: %v\n", options.RootDir, err)
	}
}

func TestPull(t *testing.T) {
	d := prepare(t)

	pullID := stringid.GenerateNonCryptoID()[:constant.DefaultIDLen]
	req := &pb.PullRequest{
		PullID:    pullID,
		ImageName: "255.255.255.255/no-repository/no-name",
	}
	stream := &controlPullServer{}
	err := d.Daemon.backend.Pull(req, stream)
	assert.ErrorContains(t, err, "failed to get the image")
	tmpClean(d)
}
