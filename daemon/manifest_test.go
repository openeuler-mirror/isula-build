// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: daisicheng
// Create: 2022-12-15
// Description: This file tests manifest interface.

package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/pkg/logger"
)

type controlManifestPushServer struct {
	grpc.ServerStream
}

func (c *controlManifestPushServer) Context() context.Context {
	return context.Background()
}

func (c *controlManifestPushServer) Send(response *pb.ManifestPushResponse) error {
	if response.Result == "error" {
		return errors.New("error happened")
	}
	return nil
}

func TestManifestCreate(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	ctx := context.TODO()
	req := &pb.ManifestCreateRequest{ManifestList: "openeuler",
		Manifests: []string{"openeuler_arrch64"}}
	_, err := d.Daemon.backend.ManifestCreate(ctx, req)
	assert.ErrorContains(t, err, "enable experimental to use manifest feature")

	d.Daemon.opts.Experimental = true
	req = &pb.ManifestCreateRequest{ManifestList: "euleros",
		Manifests: []string{"euleros_x86"}}
	_, err = d.Daemon.backend.ManifestCreate(ctx, req)
	assert.ErrorContains(t, err, "failed to get the image")

}

func TestManifestAnnotate(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	ctx := context.TODO()
	req := &pb.ManifestAnnotateRequest{ManifestList: "openeuler",
		Manifest: "openeuler_arrch64"}
	_, err := d.Daemon.backend.ManifestAnnotate(ctx, req)
	assert.ErrorContains(t, err, "enable experimental to use manifest feature")

	d.Daemon.opts.Experimental = true
	req = &pb.ManifestAnnotateRequest{ManifestList: "euleros",
		Manifest: "euleros_x86"}
	_, err = d.Daemon.backend.ManifestAnnotate(ctx, req)
	assert.ErrorContains(t, err, "not found in local store")

	options := &storage.ImageOptions{}
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	req = &pb.ManifestAnnotateRequest{ManifestList: "image",
		Manifest: "euleros_x86"}
	_, err = d.Daemon.backend.ManifestAnnotate(ctx, req)
	fmt.Println(err)
	assert.ErrorContains(t, err, "file does not exist")
}

func TestManifestInspect(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	ctx := context.TODO()
	_, err := d.Daemon.backend.ManifestInspect(ctx, &pb.ManifestInspectRequest{ManifestList: "openeuler"})
	assert.ErrorContains(t, err, "enable experimental to use manifest feature")

	d.Daemon.opts.Experimental = true
	_, err = d.Daemon.backend.ManifestInspect(ctx, &pb.ManifestInspectRequest{ManifestList: "euleros"})
	assert.ErrorContains(t, err, "not found in local store")

	options := &storage.ImageOptions{}
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	_, err = d.Daemon.backend.ManifestInspect(ctx, &pb.ManifestInspectRequest{ManifestList: "image"})
	assert.ErrorContains(t, err, "file does not exist")
}

func TestManifestPush(t *testing.T) {
	d := prepare(t)
	defer tmpClean(d)

	stream := &controlManifestPushServer{}
	req := &pb.ManifestPushRequest{ManifestList: "openeuler", Dest: "127.0.0.1/no-repository"}
	err := d.Daemon.backend.ManifestPush(req, stream)
	assert.ErrorContains(t, err, "enable experimental to use manifest feature")

	d.Daemon.opts.Experimental = true
	req = &pb.ManifestPushRequest{ManifestList: "euleros", Dest: "127.0.0.1/no-repository"}
	err = d.Daemon.backend.ManifestPush(req, stream)
	assert.ErrorContains(t, err, "not found in local store")

	options := &storage.ImageOptions{}
	_, err = d.Daemon.localStore.CreateImage(stringid.GenerateRandomID(), []string{"image"}, "", "", options)
	if err != nil {
		t.Fatalf("create image with error: %v", err)
	}
	req = &pb.ManifestPushRequest{ManifestList: "image", Dest: "127.0.0.1/no-repository"}
	err = d.Daemon.backend.ManifestPush(req, stream)
	assert.ErrorContains(t, err, "file does not exist")
}

func TestManifestPushHandler(t *testing.T) {
	ctx := context.TODO()
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(manifestPushHandlerPrint("Push Response"))
	eg.Go(manifestPushHandlerPrint(""))
	eg.Go(manifestPushHandlerPrint("error"))

	eg.Wait()
}

func manifestPushHandlerPrint(message string) func() error {
	return func() error {
		stream := &controlManifestPushServer{}
		cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)

		ctx := context.TODO()
		eg, _ := errgroup.WithContext(ctx)

		eg.Go(manifestPushMessageHandler(stream, cliLogger))
		eg.Go(func() error {
			cliLogger.Print(message)
			cliLogger.CloseContent()
			return nil
		})

		eg.Wait()

		return nil
	}
}
