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

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	_ "isula.org/isula-build/exporter/docker"
	"isula.org/isula-build/pkg/logger"
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
	if response.Response == "error" {
		return errors.New("error happened")
	}
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
	defer tmpClean(d)

	options := &storage.ImageOptions{}
	_, err := d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image:test"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}

	testcases := []struct {
		name      string
		req       *pb.PullRequest
		wantErr   bool
		errString string
	}{
		{
			name: "abnormal case with no corresponding image in local store",
			req: &pb.PullRequest{
				PullID:    stringid.GenerateNonCryptoID()[:constant.DefaultIDLen],
				ImageName: "255.255.255.255/no-repository/no-name",
			},
			wantErr:   true,
			errString: "failed to get the image",
		},
		{
			name: "normal case with image in local store",
			req: &pb.PullRequest{
				PullID:    stringid.GenerateNonCryptoID()[:constant.DefaultIDLen],
				ImageName: "image:test",
			},
			wantErr: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &controlPullServer{}

			err := d.Daemon.backend.Pull(tc.req, stream)
			if tc.wantErr == true {
				assert.ErrorContains(t, err, tc.errString)
			}
			if tc.wantErr == false {
				assert.NilError(t, err)
			}
		})
	}

}

func TestPullHandler(t *testing.T) {
	ctx := context.TODO()
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(pullHandlerPrint("Push Response"))
	eg.Go(pullHandlerPrint(""))
	eg.Go(pullHandlerPrint("error"))

	eg.Wait()
}

func pullHandlerPrint(message string) func() error {
	return func() error {
		stream := &controlPullServer{}
		cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)

		ctx := context.TODO()
		eg, _ := errgroup.WithContext(ctx)

		eg.Go(pullMessageHandler(stream, cliLogger))
		eg.Go(func() error {
			cliLogger.Print(message)
			cliLogger.CloseContent()
			return nil
		})

		eg.Wait()

		return nil
	}
}
