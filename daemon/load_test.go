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

func prepareForLoad(t *testing.T, manifest string) *fs.Dir {
	tmpDir := fs.NewDir(t, t.Name(), fs.WithFile("manifest.json", manifest))
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
		localStore: &localStore,
	}
	daemon.NewBackend()

	return tmpDir
}

func clean(dir *fs.Dir) {
	unix.Unmount(dir.Join("lib", "overlay"), 0)
	dir.Remove()
}

func TestLoad(t *testing.T) {
	manifestJson :=
		`[
			{
				"Config":"76a4dd2d5d6a18323ac8d90f959c3c8562bf592e2a559bab9b462ab600e9e5fc.json",
				"RepoTags":[
					"hello:latest"
				],
				"Layers":[
					"6eb4c21cc3fcb729a9df230ae522c1d3708ca66e5cf531713dbfa679837aa287.tar",
					"37841116ad3b1eeea972c75ab8bad05f48f721a7431924bc547fc91c9076c1c8.tar"
				]
			}
		]`
	dir := prepareForLoad(t, manifestJson)
	assert.Equal(t, dir != nil, true)
	defer clean(dir)

	path := dir.Join("load.tar")
	repoTags, err := getRepoTagFromImageTar(daemon.opts.DataRoot, path)
	assert.NilError(t, err)
	assert.Equal(t, repoTags[0][0], "hello:latest")

	req := &pb.LoadRequest{Path: path}
	stream := &controlLoadServer{}

	err = daemon.backend.Load(req, stream)
	assert.ErrorContains(t, err, "failed to get the image")
}

func TestLoadMultipleImages(t *testing.T) {
	manifestJson :=
		`[
			{
				"Config": "e4db68de4ff27c2adfea0c54bbb73a61a42f5b667c326de4d7d5b19ab71c6a3b.json",
				"RepoTags": [
				"registry.example.com/sayhello:first"
				],
				"Layers": [
				"6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a.tar"
				]
			},
			{
				"Config": "c07ddb44daa97e9e8d2d68316b296cc9343ab5f3d2babc5e6e03b80cd580478e.json",
				"RepoTags": [
				"registry.example.com/sayhello:second",
				"registry.example.com/sayhello:third"
				],
				"Layers": [
				"e7ebc6e16708285bee3917ae12bf8d172ee0d7684a7830751ab9a1c070e7a125.tar"
				]
			},
			{
				"Config": "f643c72bc25212974c16f3348b3a898b1ec1eb13ec1539e10a103e6e217eb2f1.json",
				"RepoTags": [],
				"Layers": [
				  "bacd3af13903e13a43fe87b6944acd1ff21024132aad6e74b4452d984fb1a99a.tar",
				  "9069f84dbbe96d4c50a656a05bbe6b6892722b0d1116a8f7fd9d274f4e991bf6.tar",
				  "f6253634dc78da2f2e3bee9c8063593f880dc35d701307f30f65553e0f50c18c.tar"
				]
			}
		]`
	dir := prepareForLoad(t, manifestJson)
	assert.Equal(t, dir != nil, true)
	defer clean(dir)

	path := dir.Join("load.tar")
	repoTags, err := getRepoTagFromImageTar(daemon.opts.DataRoot, path)
	assert.NilError(t, err)
	assert.Equal(t, repoTags[0][0], "registry.example.com/sayhello:first")
	assert.Equal(t, repoTags[1][0], "registry.example.com/sayhello:second")
	assert.Equal(t, repoTags[1][1], "registry.example.com/sayhello:third")
	assert.Equal(t, len(repoTags[2]), 0)

	req := &pb.LoadRequest{Path: path}
	stream := &controlLoadServer{}

	err = daemon.backend.Load(req, stream)
	assert.ErrorContains(t, err, "failed to get the image")
}
