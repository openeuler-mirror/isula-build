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
	"context"
	"os"
	"strings"

	"github.com/containers/image/v5/docker/tarfile"
	ociarchive "github.com/containers/image/v5/oci/archive"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	securejoin "github.com/cyphar/filepath-securejoin"
	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/daemon/separator"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

type singleImage struct {
	index   int
	id      string
	nameTag []string
}

// LoadOptions stores the options for image loading
type LoadOptions struct {
	LogEntry *logrus.Entry
	path     string
	format   string
	sep      separator.Loader
}

func (b *Backend) getLoadOptions(req *pb.LoadRequest) (LoadOptions, error) {
	var err error
	var opt = LoadOptions{
		path:     req.GetPath(),
		LogEntry: logrus.WithFields(logrus.Fields{"LoadID": req.GetLoadID()}),
	}

	// normal image loading
	if !req.GetSep().GetEnabled() {
		if err = util.CheckFileInfoAndSize(opt.path, constant.MaxLoadFileSize); err != nil {
			return LoadOptions{}, err
		}
		return opt, nil
	}

	// separated images loading
	opt.sep, err = separator.GetSepLoadOptions(req, opt.LogEntry, b.daemon.opts.DataRoot)
	if err != nil {
		return LoadOptions{}, err
	}

	return opt, nil
}

// Load loads the image
func (b *Backend) Load(req *pb.LoadRequest, stream pb.Control_LoadServer) error {
	logrus.WithFields(logrus.Fields{
		"LoadID": req.GetLoadID(),
	}).Info("LoadRequest received")

	var si *storage.Image

	opts, err := b.getLoadOptions(req)
	if err != nil {
		return errors.Wrap(err, "process load options failed")
	}

	defer func() {
		if tErr := os.RemoveAll(opts.sep.TmpDirRoot()); tErr != nil {
			opts.LogEntry.Warnf("Removing load tmp directory %q failed: %v", opts.sep.TmpDirRoot(), tErr)
		}
	}()

	// construct separated images
	if opts.sep.Enabled() {
		var lErr error
		if opts.path, lErr = opts.sep.LoadSeparatedImage(); lErr != nil {
			opts.LogEntry.Errorf("Load separated image for %s failed: %v", opts.sep.AppName(), lErr)
			return lErr
		}
	}

	imagesInTar, err := tryToParseImageFormatFromTarball(b.daemon.opts.DataRoot, &opts)
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

		for _, singleImage := range imagesInTar {
			_, si, err = image.ResolveFromImage(&image.PrepareImageOptions{
				Ctx:           ctx,
				FromImage:     exporter.FormatTransport(opts.format, opts.path),
				ToImage:       singleImage.id,
				SystemContext: image.GetSystemContext(),
				Store:         b.daemon.localStore,
				Reporter:      log,
				ManifestIndex: singleImage.index,
			})
			if err != nil {
				return err
			}

			originalNames, err := b.daemon.localStore.Names(si.ID)
			if err != nil {
				return err
			}
			if err = b.daemon.localStore.SetNames(si.ID, append(originalNames, singleImage.nameTag...)); err != nil {
				return err
			}

			log.Print("Loaded image as %s\n", si.ID)
			logrus.Infof("Loaded image as %s", si.ID)
		}

		return nil
	})

	if wErr := eg.Wait(); wErr != nil {
		return wErr
	}

	return nil
}

func tryToParseImageFormatFromTarball(dataRoot string, opts *LoadOptions) ([]singleImage, error) {
	// tmp dir will be removed after NewSourceFromFileWithContext
	tmpDir, err := securejoin.SecureJoin(dataRoot, constant.DataRootTmpDirPrefix)
	if err != nil {
		return nil, err
	}
	systemContext := image.GetSystemContext()
	systemContext.BigFilesTemporaryDir = tmpDir

	// try docker format loading
	imagesInTar, err := getDockerRepoTagFromImageTar(systemContext, opts.path)
	if err == nil {
		logrus.Infof("Parse image successful with %q format", constant.DockerTransport)
		opts.format = constant.DockerArchiveTransport
		return imagesInTar, nil
	}
	logrus.Warnf("Try to Parse image of docker format failed with error: %v", err)

	// try oci format loading
	imagesInTar, err = getOCIRepoTagFromImageTar(systemContext, opts.path)
	if err == nil {
		logrus.Infof("Parse image successful with %q format", constant.OCITransport)
		opts.format = constant.OCIArchiveTransport
		return imagesInTar, nil
	}
	logrus.Warnf("Try to parse image of oci format failed with error: %v", err)

	// record the last error
	return nil, errors.Wrap(err, "wrong image format detected from local tarball")
}

func getDockerRepoTagFromImageTar(systemContext *types.SystemContext, path string) ([]singleImage, error) {
	// tmp dir will be removed after NewSourceFromFileWithContext
	tarfileSource, err := tarfile.NewSourceFromFileWithContext(systemContext, path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the source of loading tar file")
	}
	defer func() {
		if cErr := tarfileSource.Close(); cErr != nil {
			logrus.Warnf("tar file source close failed: %v", cErr)
		}
	}()

	topLevelImageManifest, err := tarfileSource.LoadTarManifest()
	if err != nil || len(topLevelImageManifest) == 0 {
		return nil, errors.Errorf("failed to get the top level image manifest: %v", err)
	}

	imagesInTar := make([]singleImage, 0, len(topLevelImageManifest))
	for i, manifestItem := range topLevelImageManifest {
		imageID, err := parseConfigID(manifestItem.Config)
		if err != nil {
			return nil, err
		}
		imagesInTar = append(imagesInTar, singleImage{index: i, id: imageID, nameTag: manifestItem.RepoTags})
	}

	return imagesInTar, nil
}

func getOCIRepoTagFromImageTar(systemContext *types.SystemContext, path string) ([]singleImage, error) {
	srcRef, err := alltransports.ParseImageName(exporter.FormatTransport(constant.OCIArchiveTransport, path))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse image name of oci image format")
	}

	imageID, err := getLoadedImageID(srcRef, systemContext)
	if err != nil {
		return nil, err
	}
	tarManifest, err := ociarchive.LoadManifestDescriptorWithContext(systemContext, srcRef)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load manifest descriptor of oci image format")
	}

	// For now, we only support loading oci-archive file with one single image
	if _, ok := tarManifest.Annotations[imgspecv1.AnnotationRefName]; ok {
		return []singleImage{{0, imageID, []string{tarManifest.Annotations[imgspecv1.AnnotationRefName]}}}, nil
	}
	return []singleImage{{0, imageID, []string{}}}, nil
}

func parseConfigID(configID string) (string, error) {
	parts := strings.SplitN(configID, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("wrong config info of manifest.json")
	}

	configDigest := "sha256:" + digest.Digest(parts[0])
	if err := configDigest.Validate(); err != nil {
		return "", errors.Wrapf(err, "failed to get config info")
	}

	return "@" + configDigest.Encoded(), nil
}

func getLoadedImageID(imageRef types.ImageReference, systemContext *types.SystemContext) (string, error) {
	if imageRef == nil || systemContext == nil {
		return "", errors.New("nil image reference or system context when loading image")
	}

	newImage, err := imageRef.NewImage(context.TODO(), systemContext)
	if err != nil {
		return "", err
	}
	defer func() {
		if err = newImage.Close(); err != nil {
			logrus.Errorf("failed to close image: %v", err)
		}
	}()
	imageDigest := newImage.ConfigInfo().Digest
	if err = imageDigest.Validate(); err != nil {
		return "", errors.Wrap(err, "failed to get config info")
	}

	return "@" + imageDigest.Encoded(), nil
}
