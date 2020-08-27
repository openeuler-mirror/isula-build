// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Feiyu Yang
// Create: 2020-08-20
// Description: This file is for image load test.

package daemon

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"gotest.tools/assert"
	"gotest.tools/fs"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/store"
)

var (
	localStore store.Store
	daemon     *Daemon
)

func init() {
	reexec.Init()
}

type controlLoadServer struct {
	grpc.ServerStream
}

func (x *controlLoadServer) Send(m *pb.LoadResponse) error {
	return nil
}

func (x *controlLoadServer) Context() context.Context {
	return context.Background()
}

func prepareLoadTar(dir *fs.Dir) error {
	manifest := dir.Join("manifest.json")

	fi, err := os.Create(dir.Join("load.tar"))
	if err != nil {
		return nil
	}
	defer fi.Close()

	tw := tar.NewWriter(fi)
	defer tw.Close()

	manifestFi, err := os.Stat(manifest)
	if err != nil {
		return nil
	}

	hdr, err := tar.FileInfoHeader(manifestFi, "")
	if err != nil {
		return nil
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return nil
	}

	manifestFile, err := os.Open(manifest)
	if err != nil {
		return nil
	}

	_, err = io.Copy(tw, manifestFile)

	return err

}

func prepareForLoad(t *testing.T) *fs.Dir {
	dockerfile := `[{"Config":"76a4dd2d5d6a18323ac8d90f959c3c8562bf592e2a559bab9b462ab600e9e5fc.json",
	"RepoTags":["hello:latest"],
	"Layers":["6eb4c21cc3fcb729a9df230ae522c1d3708ca66e5cf531713dbfa679837aa287.tar",
	"37841116ad3b1eeea972c75ab8bad05f48f721a7431924bc547fc91c9076c1c8.tar"]}]`
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile("manifest.json", dockerfile))
	if err := prepareLoadTar(tmpDir); err != nil {
		tmpDir.Remove()
		return nil
	}

	opt := &Options{
		DataRoot: tmpDir.Join("lib"),
		RunRoot:  tmpDir.Join("run"),
	}
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: opt.DataRoot,
		RunRoot:  opt.RunRoot,
	})
	localStore, _ = store.GetStore()

	daemon = &Daemon{
		opts:       opt,
		localStore: localStore,
	}
	daemon.NewBackend()

	return tmpDir
}

func clean(dir *fs.Dir) {
	unix.Unmount(dir.Join("lib", "overlay"), 0)
	dir.Remove()
}

func TestLoad(t *testing.T) {
	dir := prepareForLoad(t)
	assert.Equal(t, dir != nil, true)
	defer clean(dir)

	path := dir.Join("load.tar")
	repoTags, err := getRepoTagFromImageTar(daemon.opts.DataRoot, path)
	assert.NilError(t, err)
	assert.Equal(t, repoTags[0], "hello:latest")

	req := &pb.LoadRequest{Path: path}
	stream := &controlLoadServer{}

	err = daemon.backend.Load(req, stream)
	assert.ErrorContains(t, err, "failed to get the image")
}
