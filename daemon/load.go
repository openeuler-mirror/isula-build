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
// Create: 2020-07-17
// Description: This file is used for image load command.

package daemon

import (
	"github.com/containers/image/v5/docker/tarfile"
	"github.com/containers/storage"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

// Load loads the image
func (b *Backend) Load(req *pb.LoadRequest, stream pb.Control_LoadServer) error {
	var si *storage.Image
	logrus.Info("LoadRequest received")
	if err := util.CheckLoadFile(req.Path); err != nil {
		return err
	}

	repoTags, err := getRepoTagFromImageTar(b.daemon.opts.DataRoot, req.Path)
	if err != nil {
		return err
	}

	log := logger.NewCliLogger(constant.CliLogBufferLen)
	eg, ctx := errgroup.WithContext(stream.Context())
	eg.Go(func() error {
		for c := range log.GetContent() {
			if serr := stream.Send(&pb.LoadResponse{
				Log: c,
			}); serr != nil {
				return serr
			}
		}
		return nil
	})

	eg.Go(func() error {
		defer log.CloseContent()
		_, si, err = image.ResolveFromImage(&image.PrepareImageOptions{
			Ctx:           ctx,
			FromImage:     "docker-archive:" + req.Path,
			SystemContext: image.GetSystemContext(),
			Store:         b.daemon.localStore,
			Reporter:      log,
		})
		if err != nil {
			return err
		}

		if serr := b.daemon.localStore.SetNames(si.ID, repoTags); serr != nil {
			return serr
		}
		log.Print("Loaded image as %s\n", si.ID)
		return nil
	})

	if werr := eg.Wait(); werr != nil {
		return werr
	}
	logrus.Infof("Loaded image as %s", si.ID)

	return nil
}

func getRepoTagFromImageTar(dataRoot, path string) ([]string, error) {
	// tmp dir will be removed after NewSourceFromFileWithContext
	tmpDir, err := securejoin.SecureJoin(dataRoot, dataRootTmpDirPrefix)
	if err != nil {
		return nil, err
	}
	systemContext := image.GetSystemContext()
	systemContext.BigFilesTemporaryDir = tmpDir

	tarfileSource, err := tarfile.NewSourceFromFileWithContext(systemContext, path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the source of loading tar file")
	}

	topLevelImageManifest, err := tarfileSource.LoadTarManifest()
	if err != nil || len(topLevelImageManifest) == 0 {
		return nil, errors.Errorf("failed to get the top level image manifest: %v", err)
	}

	return topLevelImageManifest[0].RepoTags, nil
}
