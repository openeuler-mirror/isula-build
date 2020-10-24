// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu
// Create: 2020-07-22
// Description: This file is "import" command for backend

package daemon

import (
	"io"
	"os"
	"path/filepath"

	cp "github.com/containers/image/v5/copy"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/tarball"
	"github.com/containers/image/v5/transports"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

// Import an image from a tarball
func (b *Backend) Import(serv pb.Control_ImportServer) error {
	logrus.Info("ImportRequest received")

	localStore := b.daemon.localStore
	buf := make([]byte, 0, constant.BufferSize)
	reference := ""
	for {
		msg, ierr := serv.Recv()
		if ierr == io.EOF {
			break
		}
		if ierr != nil {
			return ierr
		}
		if msg == nil {
			return errors.New("import failed, receive nil msg")
		}
		reference = msg.Reference
		buf = append(buf, msg.Data...)
	}

	_, reference, err := dockerfile.CheckAndExpandTag(reference)
	if err != nil {
		return err
	}
	logrus.Infof("Received and import image as %q", reference)
	srcRef, err := tarball.NewReference([]string{"-"}, buf)
	if err != nil {
		return err
	}

	tmpName := stringid.GenerateRandomID() + "-import-tmp"
	dstRef, err := is.Transport.ParseStoreReference(localStore, tmpName)
	if err != nil {
		return err
	}
	policyContext, err := dockerfile.GetPolicyContext()
	if err != nil {
		return err
	}
	log := logger.NewCliLogger(constant.CliLogBufferLen)
	imageCopyOptions := image.NewImageCopyOptions(log)
	tmpDir := filepath.Join(b.daemon.opts.DataRoot, "tmp")
	if err = os.MkdirAll(tmpDir, constant.DefaultRootDirMode); err != nil {
		return err
	}
	defer func() {
		if rerr := os.Remove(tmpDir); rerr != nil {
			logrus.Warnf("Remove tmp dir %q failed", rerr)
		}
	}()
	imageCopyOptions.SourceCtx.BigFilesTemporaryDir = tmpDir
	imageCopyOptions.DestinationCtx.BigFilesTemporaryDir = tmpDir

	eg, ctx := errgroup.WithContext(serv.Context())
	eg.Go(func() error {
		for c := range log.GetContent() {
			if serr := serv.Send(&pb.ImportResponse{
				Log: c,
			}); serr != nil {
				return serr
			}
		}
		return nil
	})

	var imageID string
	eg.Go(func() error {
		defer log.CloseContent()
		if _, err = cp.Image(ctx, policyContext, dstRef, srcRef, imageCopyOptions); err != nil {
			return err
		}
		img, err := is.Transport.GetStoreImage(localStore, dstRef)
		if err != nil {
			return errors.Wrapf(err, "error locating image %q in local storage after import", transports.ImageName(dstRef))
		}
		imageID = img.ID
		img.Names = append(img.Names, reference)
		newNames := util.CopyStringsWithoutSpecificElem(img.Names, tmpName)
		if err = localStore.SetNames(img.ID, newNames); err != nil {
			return errors.Wrapf(err, "failed to prune temporary name from image %q", imageID)
		}

		log.Print("Import success with image id: %q\n", imageID)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}
	logrus.Infof("Import success with image id: %q", imageID)

	return nil
}
