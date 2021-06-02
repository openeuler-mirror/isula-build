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
	ociarchive "github.com/containers/image/v5/oci/archive"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	securejoin "github.com/cyphar/filepath-securejoin"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

type loadOptions struct {
	path   string
	format string
}

func (b *Backend) getLoadOptions(req *pb.LoadRequest) loadOptions {
	return loadOptions{
		path: req.GetPath(),
	}
}

// Load loads the image
func (b *Backend) Load(req *pb.LoadRequest, stream pb.Control_LoadServer) error {
	logrus.Info("LoadRequest received")

	var (
		si       *storage.Image
		repoTags [][]string
		err      error
	)
	opts := b.getLoadOptions(req)

	if cErr := util.CheckLoadFile(req.Path); cErr != nil {
		return cErr
	}

	repoTags, err = tryToParseImageFormatFromTarball(b.daemon.opts.DataRoot, &opts)
	if err != nil {
		return err
	}

	log := logger.NewCliLogger(constant.CliLogBufferLen)
	eg, ctx := errgroup.WithContext(stream.Context())
	eg.Go(func() error {
		for c := range log.GetContent() {
			if sErr := stream.Send(&pb.LoadResponse{
				Log: c,
			}); sErr != nil {
				return sErr
			}
		}
		return nil
	})

	eg.Go(func() error {
		defer log.CloseContent()

		for index, nameAndTag := range repoTags {
			_, si, err = image.ResolveFromImage(&image.PrepareImageOptions{
				Ctx:           ctx,
				FromImage:     exporter.FormatTransport(opts.format, opts.path),
				SystemContext: image.GetSystemContext(),
				Store:         b.daemon.localStore,
				Reporter:      log,
				ManifestIndex: index,
			})
			if err != nil {
				return err
			}

			if sErr := b.daemon.localStore.SetNames(si.ID, nameAndTag); sErr != nil {
				return sErr
			}

			log.Print("Loaded image as %s\n", si.ID)
		}

		return nil
	})

	if wErr := eg.Wait(); wErr != nil {
		return wErr
	}
	logrus.Infof("Loaded image as %s", si.ID)

	return nil
}

func tryToParseImageFormatFromTarball(dataRoot string, opts *loadOptions) ([][]string, error) {
	var (
		allRepoTags [][]string
		err         error
	)

	// tmp dir will be removed after NewSourceFromFileWithContext
	tmpDir, err := securejoin.SecureJoin(dataRoot, dataRootTmpDirPrefix)
	if err != nil {
		return nil, err
	}
	systemContext := image.GetSystemContext()
	systemContext.BigFilesTemporaryDir = tmpDir

	allRepoTags, err = getDockerRepoTagFromImageTar(systemContext, opts.path)
	if err == nil {
		logrus.Infof("Parse image successful with %q format", constant.DockerTransport)
		opts.format = constant.DockerArchiveTransport
		return allRepoTags, nil
	}
	logrus.Warnf("Try to Parse image of docker format failed with error: %v", err)

	allRepoTags, err = getOCIRepoTagFromImageTar(systemContext, opts.path)
	if err == nil {
		logrus.Infof("Parse image successful with %q format", constant.OCITransport)
		opts.format = constant.OCIArchiveTransport
		return allRepoTags, nil
	}
	logrus.Warnf("Try to parse image of oci format failed with error: %v", err)

	// record the last error
	return nil, errors.Wrap(err, "wrong image format detected from local tarball")
}

func getDockerRepoTagFromImageTar(systemContext *types.SystemContext, path string) ([][]string, error) {
	// tmp dir will be removed after NewSourceFromFileWithContext
	tarfileSource, err := tarfile.NewSourceFromFileWithContext(systemContext, path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the source of loading tar file")
	}

	topLevelImageManifest, err := tarfileSource.LoadTarManifest()
	if err != nil || len(topLevelImageManifest) == 0 {
		return nil, errors.Wrapf(err, "failed to get the top level image manifest")
	}

	var allRepoTags [][]string
	for _, manifestItem := range topLevelImageManifest {
		allRepoTags = append(allRepoTags, manifestItem.RepoTags)
	}

	return allRepoTags, nil
}

func getOCIRepoTagFromImageTar(systemContext *types.SystemContext, path string) ([][]string, error) {
	var (
		err error
	)

	srcRef, err := alltransports.ParseImageName(exporter.FormatTransport(constant.OCIArchiveTransport, path))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image name of oci image format")
	}

	tarManifest, err := ociarchive.LoadManifestDescriptorWithContext(systemContext, srcRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load manifest descriptor of oci image format")
	}

	// For now, we only support load single image in archive file
	if _, ok := tarManifest.Annotations[imgspecv1.AnnotationRefName]; ok {
		return [][]string{{tarManifest.Annotations[imgspecv1.AnnotationRefName]}}, nil
	}

	return [][]string{{}}, nil
}
