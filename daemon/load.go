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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/docker/tarfile"
	ociarchive "github.com/containers/image/v5/oci/archive"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	securejoin "github.com/cyphar/filepath-securejoin"
	digest "github.com/opencontainers/go-digest"
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

const (
	tmpBaseDirName         = "base"
	tmpAppDirName          = "app"
	tmpLibDirName          = "lib"
	unionCompressedTarName = "all.tar.gz"
)

type loadImageTmpDir struct {
	app  string
	base string
	lib  string
	root string
}

type separatorLoad struct {
	log       *logrus.Entry
	tmpDir    loadImageTmpDir
	info      tarballInfo
	appName   string
	basePath  string
	appPath   string
	libPath   string
	dir       string
	skipCheck bool
	enabled   bool
}

type loadOptions struct {
	logEntry *logrus.Entry
	path     string
	format   string
	sep      separatorLoad
}

type singleImage struct {
	index   int
	id      string
	nameTag []string
}

func (b *Backend) getLoadOptions(req *pb.LoadRequest) (loadOptions, error) {
	var opt = loadOptions{
		path: req.GetPath(),
		sep: separatorLoad{
			appName:   req.GetSep().GetApp(),
			basePath:  req.GetSep().GetBase(),
			libPath:   req.GetSep().GetLib(),
			dir:       req.GetSep().GetDir(),
			skipCheck: req.GetSep().GetSkipCheck(),
			enabled:   req.GetSep().GetEnabled(),
		},
		logEntry: logrus.WithFields(logrus.Fields{"LoadID": req.GetLoadID()}),
	}

	// normal loadOptions
	if !opt.sep.enabled {
		if err := util.CheckLoadFile(opt.path); err != nil {
			return loadOptions{}, err
		}
		return opt, nil
	}

	// load separated images
	// log is used for sep methods
	opt.sep.log = opt.logEntry
	tmpRoot := filepath.Join(b.daemon.opts.DataRoot, filepath.Join(dataRootTmpDirPrefix, req.GetLoadID()))
	opt.sep.tmpDir.root = tmpRoot
	opt.sep.tmpDir.base = filepath.Join(tmpRoot, tmpBaseDirName)
	opt.sep.tmpDir.app = filepath.Join(tmpRoot, tmpAppDirName)
	opt.sep.tmpDir.lib = filepath.Join(tmpRoot, tmpLibDirName)

	// check image name and add "latest" tag if not present
	_, appImgName, err := image.GetNamedTaggedReference(opt.sep.appName)
	if err != nil {
		return loadOptions{}, err
	}
	opt.sep.appName = appImgName

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
		if tErr := os.RemoveAll(opts.sep.tmpDir.root); tErr != nil {
			opts.logEntry.Warnf("Removing load tmp directory %q failed: %v", opts.sep.tmpDir.root, tErr)
		}
	}()

	// construct separated images
	if opts.sep.enabled {
		if lErr := loadSeparatedImage(&opts); lErr != nil {
			opts.logEntry.Errorf("Load separated image for %s failed: %v", opts.sep.appName, lErr)
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

func tryToParseImageFormatFromTarball(dataRoot string, opts *loadOptions) ([]singleImage, error) {
	// tmp dir will be removed after NewSourceFromFileWithContext
	tmpDir, err := securejoin.SecureJoin(dataRoot, dataRootTmpDirPrefix)
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

func loadSeparatedImage(opt *loadOptions) error {
	s := &opt.sep
	s.log.Infof("Starting load separated image %s", s.appName)

	// load manifest file to get tarball info
	if err := s.getTarballInfo(); err != nil {
		return errors.Wrap(err, "failed to get tarball info")
	}
	if err := s.constructTarballInfo(); err != nil {
		return err
	}
	// checksum for image tarballs
	if err := s.tarballCheckSum(); err != nil {
		return err
	}
	// process image tarballs and get final constructed image tarball
	tarPath, err := s.processTarballs()
	if err != nil {
		return err
	}
	opt.path = tarPath

	return nil
}

func (s *separatorLoad) getTarballInfo() error {
	manifest, err := securejoin.SecureJoin(s.dir, manifestFile)
	if err != nil {
		return errors.Wrap(err, "join manifest file path failed")
	}

	var t = make(map[string]tarballInfo, 1)
	if err = util.LoadJSONFile(manifest, &t); err != nil {
		return errors.Wrap(err, "load manifest file failed")
	}

	tarball, ok := t[s.appName]
	if !ok {
		return errors.Errorf("failed to find app image %s", s.appName)
	}
	s.info = tarball

	return nil
}

func (s *separatorLoad) constructTarballInfo() (err error) {
	s.log.Infof("Construct image tarball info for %s", s.appName)
	// fill up path for separator
	// this case should not happened since client side already check this flag
	if len(s.appName) == 0 {
		return errors.New("app image name should not be empty")
	}
	s.appPath, err = securejoin.SecureJoin(s.dir, s.info.AppTarName)
	if err != nil {
		return err
	}

	if len(s.basePath) == 0 {
		if len(s.info.BaseTarName) == 0 {
			return errors.Errorf("base image %s tarball can not be empty", s.info.BaseImageName)
		}
		s.log.Info("Base image path is empty, use path from manifest")
		s.basePath, err = securejoin.SecureJoin(s.dir, s.info.BaseTarName)
		if err != nil {
			return err
		}
	}
	if len(s.libPath) == 0 && len(s.info.LibTarName) != 0 {
		s.log.Info("Lib image path is empty, use path from manifest")
		s.libPath, err = securejoin.SecureJoin(s.dir, s.info.LibTarName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *separatorLoad) tarballCheckSum() error {
	if s.skipCheck {
		s.log.Info("Skip checksum for tarballs")
		return nil
	}

	type checkInfo struct {
		path       string
		hash       string
		str        string
		canBeEmpty bool
	}
	checkLen := 3
	var checkList = make([]checkInfo, 0, checkLen)
	checkList = append(checkList, checkInfo{path: s.basePath, hash: s.info.BaseHash, canBeEmpty: false, str: "base image"})
	checkList = append(checkList, checkInfo{path: s.libPath, hash: s.info.LibHash, canBeEmpty: true, str: "lib image"})
	checkList = append(checkList, checkInfo{path: s.appPath, hash: s.info.AppHash, canBeEmpty: false, str: "app image"})
	for _, p := range checkList {
		if len(p.path) == 0 && !p.canBeEmpty {
			return errors.Errorf("%s tarball path can not be empty", p.str)
		}
		if len(p.path) != 0 {
			if err := util.CheckSum(p.path, p.hash); err != nil {
				return errors.Wrapf(err, "check sum for file %q failed", p.path)
			}
		}
	}

	return nil
}

func (s *separatorLoad) processTarballs() (string, error) {
	if err := s.unpackTarballs(); err != nil {
		return "", err
	}

	if err := s.reconstructImage(); err != nil {
		return "", err
	}

	// pack app image to tarball
	tarPath := filepath.Join(s.tmpDir.root, unionCompressedTarName)
	if err := util.PackFiles(s.tmpDir.base, tarPath, archive.Gzip, true); err != nil {
		return "", err
	}

	return tarPath, nil
}

func (s *separatorLoad) unpackTarballs() error {
	if err := s.makeTempDir(); err != nil {
		return errors.Wrap(err, "failed to make temporary directories")
	}

	type unpackInfo struct{ path, dir, str string }
	unpackLen := 3
	var unpackList = make([]unpackInfo, 0, unpackLen)
	unpackList = append(unpackList, unpackInfo{path: s.basePath, dir: s.tmpDir.base, str: "base image"})
	unpackList = append(unpackList, unpackInfo{path: s.appPath, dir: s.tmpDir.app, str: "app image"})
	unpackList = append(unpackList, unpackInfo{path: s.libPath, dir: s.tmpDir.lib, str: "lib image"})

	for _, p := range unpackList {
		if len(p.path) != 0 {
			if err := util.UnpackFile(p.path, p.dir, archive.Gzip, false); err != nil {
				return errors.Wrapf(err, "unpack %s tarball %q failed", p.str, p.path)
			}
		}
	}

	return nil
}

func (s *separatorLoad) reconstructImage() error {
	files, err := ioutil.ReadDir(s.tmpDir.app)
	if err != nil {
		return err
	}

	for _, f := range files {
		src := filepath.Join(s.tmpDir.app, f.Name())
		dest := filepath.Join(s.tmpDir.base, f.Name())
		if err := os.Rename(src, dest); err != nil {
			return errors.Wrapf(err, "reconstruct app file %q failed", s.info.AppTarName)
		}
	}

	if len(s.libPath) != 0 {
		files, err := ioutil.ReadDir(s.tmpDir.lib)
		if err != nil {
			return err
		}

		for _, f := range files {
			src := filepath.Join(s.tmpDir.lib, f.Name())
			dest := filepath.Join(s.tmpDir.base, f.Name())
			if err := os.Rename(src, dest); err != nil {
				return errors.Wrapf(err, "reconstruct lib file %q failed", s.info.LibTarName)
			}
		}
	}

	return nil
}

func (s *separatorLoad) makeTempDir() error {
	dirs := []string{s.tmpDir.root, s.tmpDir.app, s.tmpDir.base, s.tmpDir.lib}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
			return err
		}
	}

	return nil
}
