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
	"os"
	"path/filepath"

	cp "github.com/containers/image/v5/copy"
	dockerref "github.com/containers/image/v5/docker/reference"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/tarball"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/types"
	securejoin "github.com/cyphar/filepath-securejoin"
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
func (b *Backend) Import(req *pb.ImportRequest, stream pb.Control_ImportServer) error {
	var (
		srcRef     types.ImageReference
		ctx        = stream.Context()
		localStore = b.daemon.localStore
		source     = req.Source
		reference  = req.Reference
		importID   = req.ImportID
		tmpDir     string
	)
	logEntry := logrus.WithFields(logrus.Fields{"ImportID": importID})
	logEntry.Info("ImportRequest received")

	if reference != "" {
		if _, err := dockerref.Parse(reference); err != nil {
			return err
		}
	}

	tmpName := importID + "-import-tmp"
	dstRef, err := is.Transport.ParseStoreReference(localStore, tmpName)
	if err != nil {
		logEntry.Error(err)
		return err
	}
	_, reference, err = dockerfile.CheckAndExpandTag(reference)
	if err != nil {
		logEntry.Error(err)
		return err
	}
	logEntry.Infof("Received and import image as %q", reference)
	srcRef, err = tarball.NewReference([]string{source}, nil)
	if err != nil {
		logEntry.Error(err)
		return err
	}

	policyContext, err := dockerfile.GetPolicyContext()
	if err != nil {
		logEntry.Error(err)
		return err
	}
	defer func() {
		if err = policyContext.Destroy(); err != nil {
			logEntry.Debugf("Error destroying signature policy context: %v", err)
		}
	}()

	log := logger.NewCliLogger(constant.CliLogBufferLen)
	imageCopyOptions := image.NewImageCopyOptions(log)
	tmpDir, err = securejoin.SecureJoin(b.daemon.opts.DataRoot, filepath.Join(dataRootTmpDirPrefix, importID))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(tmpDir, constant.DefaultRootDirMode); err != nil {
		logEntry.Error(err)
		return err
	}
	defer func() {
		if rErr := os.RemoveAll(tmpDir); rErr != nil {
			logEntry.Errorf("Failed to remove import temporary dir %q, err: %v", filepath.Join(dataRootTmpDirPrefix, importID), rErr)
		}
	}()

	imageCopyOptions.SourceCtx.BigFilesTemporaryDir = tmpDir
	imageCopyOptions.DestinationCtx.BigFilesTemporaryDir = tmpDir
	logEntry.Debugf("Using path %q as import workspace", tmpDir)

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for c := range log.GetContent() {
			if sErr := stream.Send(&pb.ImportResponse{
				Log: c,
			}); sErr != nil {
				return sErr
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
		if reference != "" {
			img.Names = append(img.Names, reference)
		}
		newNames := util.CopyStringsWithoutSpecificElem(img.Names, tmpName)
		if err = localStore.SetNames(img.ID, newNames); err != nil {
			return errors.Wrapf(err, "failed to prune temporary name from image %q", imageID)
		}

		log.Print("Import success with image id: %q\n", imageID)
		return nil
	})

	if err := eg.Wait(); err != nil {
		logEntry.Error(err)
		return err
	}
	logEntry.Infof("Import success with image id: %q", imageID)

	return nil
}
