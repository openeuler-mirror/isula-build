// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-07-31
// Description: This file is "save" command for backend

package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/exporter"
	savedocker "isula.org/isula-build/exporter/docker/archive"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const (
	manifestDataFile     = "manifest.json"
	manifestFile         = "manifest"
	repositoriesFile     = "repositories"
	baseTarNameSuffix    = "_base_image.tar.gz"
	appTarNameSuffix     = "_app_image.tar.gz"
	libTarNameSuffix     = "_lib_image.tar.gz"
	untarTempDirName     = "untar"
	baseUntarTempDirName = "base_images"
	appUntarTempDirName  = "app_images"
	libUntarTempDirName  = "lib_images"
	unionTarName         = "all.tar"
	layerTarName         = "layer.tar"
	tarSuffix            = ".tar"
)

type savedImage struct {
	exist bool
	tags  []reference.NamedTagged
}

type saveOptions struct {
	sysCtx            *types.SystemContext
	localStore        *store.Store
	logger            *logger.Logger
	logEntry          *logrus.Entry
	saveID            string
	format            string
	outputPath        string
	oriImgList        []string
	finalImageOrdered []string
	finalImageSet     map[string]*savedImage
	sep               separatorSave
}

type separatorSave struct {
	renameData []renames
	tmpDir     imageTmpDir
	log        *logrus.Entry
	base       string
	lib        string
	dest       string
	enabled    bool
}

type renames struct {
	Name   string `json:"name"`
	Rename string `json:"rename"`
}

type imageTmpDir struct {
	app   string
	base  string
	lib   string
	untar string
	root  string
}

type layer struct {
	all  []string
	base []string
	lib  []string
	app  []string
}

type imageInfo struct {
	layers   layer
	repoTags []string
	config   string
	name     string
	tag      string
	nameTag  string
	topLayer string
}

// imageManifest return image's manifest info
type imageManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
	// Not shown in the json file
	HashMap map[string]string `json:"-"`
}

type imageLayersMap map[string]string

type tarballInfo struct {
	AppTarName    string   `json:"app"`
	AppHash       string   `json:"appHash"`
	AppLayers     []string `json:"appLayers"`
	LibTarName    string   `json:"lib"`
	LibHash       string   `json:"libHash"`
	LibImageName  string   `json:"libImageName"`
	LibLayers     []string `json:"libLayers"`
	BaseTarName   string   `json:"base"`
	BaseHash      string   `json:"baseHash"`
	BaseImageName string   `json:"baseImageName"`
	BaseLayers    []string `json:"baseLayer"`
}

func (b *Backend) getSaveOptions(req *pb.SaveRequest) (saveOptions, error) {
	var sep = separatorSave{
		base:    req.GetSep().GetBase(),
		lib:     req.GetSep().GetLib(),
		dest:    req.GetSep().GetDest(),
		log:     logrus.WithFields(logrus.Fields{"SaveID": req.GetSaveID()}),
		enabled: req.GetSep().GetEnabled(),
	}

	var opt = saveOptions{
		sysCtx:            image.GetSystemContext(),
		localStore:        b.daemon.localStore,
		saveID:            req.GetSaveID(),
		format:            req.GetFormat(),
		oriImgList:        req.GetImages(),
		finalImageOrdered: make([]string, 0),
		finalImageSet:     make(map[string]*savedImage),
		outputPath:        req.GetPath(),
		logger:            logger.NewCliLogger(constant.CliLogBufferLen),
		logEntry:          logrus.WithFields(logrus.Fields{"SaveID": req.GetSaveID(), "Format": req.GetFormat()}),
		sep:               sep,
	}
	// normal save
	if !sep.enabled {
		return opt, nil
	}

	// save separated image
	tmpRoot := filepath.Join(b.daemon.opts.DataRoot, filepath.Join(dataRootTmpDirPrefix, req.GetSaveID()))
	untar := filepath.Join(tmpRoot, untarTempDirName)
	appDir := filepath.Join(tmpRoot, appUntarTempDirName)
	baseDir := filepath.Join(tmpRoot, baseUntarTempDirName)
	libDir := filepath.Join(tmpRoot, libUntarTempDirName)

	opt.sep.tmpDir = imageTmpDir{
		app:   appDir,
		base:  baseDir,
		lib:   libDir,
		untar: untar,
		root:  tmpRoot,
	}
	opt.outputPath = filepath.Join(untar, unionTarName)
	renameFile := req.GetSep().GetRename()
	if len(renameFile) != 0 {
		var reName []renames
		if err := util.LoadJSONFile(renameFile, &reName); err != nil {
			return saveOptions{}, err
		}
		opt.sep.renameData = reName
	}

	return opt, nil
}

// Save receives a save request and save the image(s) into tarball
func (b *Backend) Save(req *pb.SaveRequest, stream pb.Control_SaveServer) error {
	logrus.WithFields(logrus.Fields{
		"SaveID": req.GetSaveID(),
		"Format": req.GetFormat(),
	}).Info("SaveRequest received")

	var err error
	opts, err := b.getSaveOptions(req)
	if err != nil {
		return errors.Wrap(err, "process save options failed")
	}

	if err = checkFormat(&opts); err != nil {
		return err
	}
	if err = filterImageName(&opts); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rErr := os.Remove(opts.outputPath); rErr != nil && !os.IsNotExist(rErr) {
				opts.logEntry.Warnf("Removing save output tarball %q failed: %v", opts.outputPath, rErr)
			}
		}
	}()

	ctx := context.WithValue(stream.Context(), util.LogFieldKey(util.LogKeySessionID), opts.saveID)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(exportHandler(ctx, &opts))
	eg.Go(messageHandler(stream, opts.logger))

	if err = eg.Wait(); err != nil {
		opts.logEntry.Warnf("Save stream closed with: %v", err)
		return err
	}

	// separatorSave found
	if opts.sep.enabled {
		return separateImage(opts)
	}

	return nil
}

func exportHandler(ctx context.Context, opts *saveOptions) func() error {
	return func() error {
		defer func() {
			opts.logger.CloseContent()
			if savedocker.DockerArchiveExporter.GetArchiveWriter(opts.saveID) != nil {
				if cErr := savedocker.DockerArchiveExporter.GetArchiveWriter(opts.saveID).Close(); cErr != nil {
					opts.logEntry.Errorf("Close archive writer failed: %v", cErr)
				}
				savedocker.DockerArchiveExporter.RemoveArchiveWriter(opts.saveID)
			}
		}()

		if err := os.MkdirAll(filepath.Dir(opts.outputPath), constant.DefaultRootFileMode); err != nil {
			return err
		}
		for _, imageID := range opts.finalImageOrdered {
			copyCtx := *opts.sysCtx
			if opts.format == constant.DockerArchiveTransport {
				// It's ok for DockerArchiveAdditionalTags == nil, as a result, no additional tags will be appended to the final archive file.
				copyCtx.DockerArchiveAdditionalTags = opts.finalImageSet[imageID].tags
			}

			exOpts := exporter.ExportOptions{
				Ctx:           ctx,
				SystemContext: &copyCtx,
				ExportID:      opts.saveID,
				ReportWriter:  opts.logger,
			}

			if err := exporter.Export(imageID, exporter.FormatTransport(opts.format, opts.outputPath),
				exOpts, opts.localStore); err != nil {
				opts.logEntry.Errorf("Save image %q in format %q failed: %v", imageID, opts.format, err)
				return errors.Wrapf(err, "save image %q in format %q failed", imageID, opts.format)
			}
		}

		return nil
	}
}

func messageHandler(stream pb.Control_SaveServer, cliLogger *logger.Logger) func() error {
	return func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err := stream.Send(&pb.SaveResponse{
				Log: content,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}

func checkFormat(opts *saveOptions) error {
	switch opts.format {
	case constant.DockerTransport:
		opts.format = constant.DockerArchiveTransport
	case constant.OCITransport:
		opts.format = constant.OCIArchiveTransport
	default:
		return errors.New("wrong image format provided")
	}

	return nil
}

func filterImageName(opts *saveOptions) error {
	if opts.format == constant.OCIArchiveTransport {
		opts.finalImageOrdered = opts.oriImgList
		return nil
	}

	visitedImage := make(map[string]bool)
	for _, imageName := range opts.oriImgList {
		if _, exists := visitedImage[imageName]; exists {
			continue
		}
		visitedImage[imageName] = true

		_, img, err := image.FindImage(opts.localStore, imageName)
		if err != nil {
			return errors.Wrapf(err, "filter image name failed when finding image name %q", imageName)
		}

		finalImage, ok := opts.finalImageSet[img.ID]
		if !ok {
			finalImage = &savedImage{exist: true}
			finalImage.tags = []reference.NamedTagged{}
			opts.finalImageOrdered = append(opts.finalImageOrdered, img.ID)
		}

		if !strings.HasPrefix(img.ID, imageName) {
			tagged, _, err := image.GetNamedTaggedReference(imageName)
			if err != nil {
				return errors.Wrapf(err, "get named tagged reference failed when saving image %q", imageName)
			}
			finalImage.tags = append(finalImage.tags, tagged)
		}
		opts.finalImageSet[img.ID] = finalImage
	}

	return nil
}

func getLayerHashFromStorage(store *store.Store, name string) ([]string, error) {
	if len(name) == 0 {
		return nil, nil
	}
	_, img, err := image.FindImage(store, name)
	if err != nil {
		return nil, err
	}

	layer, err := store.Layer(img.TopLayer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get top layer for image %s", name)
	}

	var layers []string
	// add each layer in the layers until reach the root layer
	for layer != nil {
		fields := strings.Split(layer.UncompressedDigest.String(), ":")
		if len(fields) != 2 {
			return nil, errors.Errorf("error format of layer of image %s", name)
		}
		layers = append(layers, fields[1])
		if layer.Parent == "" {
			break
		}
		layer, err = store.Layer(layer.Parent)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read layer %q", layer.Parent)
		}
	}

	return layers, nil
}

// process physic file
func (s *separatorSave) constructLayerMap() (map[string]string, error) {
	path := s.tmpDir.untar
	files, rErr := ioutil.ReadDir(path)
	if rErr != nil {
		return nil, rErr
	}

	var layerMap = make(map[string]string, len(files))
	// process layer's file
	for _, file := range files {
		if file.IsDir() {
			layerFile := filepath.Join(path, file.Name(), layerTarName)
			oriFile, err := os.Readlink(layerFile)
			if err != nil {
				return nil, err
			}
			physicFile := filepath.Join(path, file.Name(), oriFile)
			layerMap[filepath.Base(physicFile)] = filepath.Join(file.Name(), layerTarName)
			if err := os.Rename(physicFile, layerFile); err != nil {
				return nil, err
			}
		}
	}

	return layerMap, nil
}

func getLayerHashFromTar(layerMap map[string]string, layer []string) map[string]string {
	hashMap := make(map[string]string, len(layer))
	// first reverse map since it's <k-v> is unique
	revMap := make(map[string]string, len(layerMap))
	for k, v := range layerMap {
		revMap[v] = k
	}
	for _, l := range layer {
		if v, ok := revMap[l]; ok {
			// format is like xxx(hash): xxx/layer.tar
			hashMap[strings.TrimSuffix(v, tarSuffix)] = l
		}
	}

	return hashMap
}

func (s *separatorSave) adjustLayers() ([]imageManifest, error) {
	s.log.Info("Adjusting layers for saving separated image")

	layerMap, err := s.constructLayerMap()
	if err != nil {
		s.log.Errorf("Process layers failed: %v", err)
		return nil, err
	}

	// process manifest file
	var man []imageManifest
	if lErr := util.LoadJSONFile(filepath.Join(s.tmpDir.untar, manifestDataFile), &man); lErr != nil {
		return nil, lErr
	}

	for i, img := range man {
		layers := make([]string, len(img.Layers))
		for i, layer := range img.Layers {
			layers[i] = layerMap[layer]
		}
		man[i].Layers = layers
		man[i].HashMap = getLayerHashFromTar(layerMap, layers)
	}
	buf, err := json.Marshal(&man)
	if err != nil {
		return nil, err
	}
	if err := ioutils.AtomicWriteFile(manifestFile, buf, constant.DefaultSharedFileMode); err != nil {
		return nil, err
	}

	return man, nil
}

func separateImage(opt saveOptions) error {
	s := &opt.sep
	s.log.Infof("Start saving separated images %v", opt.oriImgList)
	var errList []error

	if err := os.MkdirAll(s.dest, constant.DefaultRootDirMode); err != nil {
		return err
	}

	defer func() {
		if tErr := os.RemoveAll(s.tmpDir.root); tErr != nil && !os.IsNotExist(tErr) {
			s.log.Warnf("Removing save tmp directory %q failed: %v", s.tmpDir.root, tErr)
		}
		if len(errList) != 0 {
			if rErr := os.RemoveAll(s.dest); rErr != nil && !os.IsNotExist(rErr) {
				s.log.Warnf("Removing save dest directory %q failed: %v", s.dest, rErr)
			}
		}
	}()
	if err := util.UnpackFile(opt.outputPath, s.tmpDir.untar, archive.Gzip, true); err != nil {
		errList = append(errList, err)
		return errors.Wrapf(err, "unpack %q failed", opt.outputPath)
	}
	manifest, err := s.adjustLayers()
	if err != nil {
		errList = append(errList, err)
		return errors.Wrap(err, "adjust layers failed")
	}

	imgInfos, err := s.constructImageInfos(manifest, opt.localStore)
	if err != nil {
		errList = append(errList, err)
		return errors.Wrap(err, "process image infos failed")
	}

	if err := s.processImageLayers(imgInfos); err != nil {
		errList = append(errList, err)
		return err
	}

	return nil
}

func (s *separatorSave) processImageLayers(imgInfos map[string]imageInfo) error {
	s.log.Info("Processing image layers")
	var (
		tarballs      = make(map[string]tarballInfo)
		baseImagesMap = make(imageLayersMap, 1)
		libImagesMap  = make(imageLayersMap, 1)
		appImagesMap  = make(imageLayersMap, 1)
	)
	var sortedKey []string
	for k := range imgInfos {
		sortedKey = append(sortedKey, k)
	}
	sort.Strings(sortedKey)
	for _, k := range sortedKey {
		info := imgInfos[k]
		if err := s.clearDirs(true); err != nil {
			return errors.Wrap(err, "clear tmp dirs failed")
		}
		var t tarballInfo
		// process base
		if err := info.processBaseImg(s, baseImagesMap, &t); err != nil {
			return errors.Wrapf(err, "process base images %s failed", info.nameTag)
		}
		// process lib
		if err := info.processLibImg(s, libImagesMap, &t); err != nil {
			return errors.Wrapf(err, "process lib images %s failed", info.nameTag)
		}
		// process app
		if err := info.processAppImg(s, appImagesMap, &t); err != nil {
			return errors.Wrapf(err, "process app images %s failed", info.nameTag)
		}
		tarballs[info.nameTag] = t
	}
	buf, err := json.Marshal(&tarballs)
	if err != nil {
		return err
	}
	// manifest file
	manifestFile := filepath.Join(s.dest, manifestFile)
	if err := ioutils.AtomicWriteFile(manifestFile, buf, constant.DefaultRootFileMode); err != nil {
		return err
	}

	s.log.Info("Save separated image succeed")
	return nil
}

func (s *separatorSave) clearDirs(reCreate bool) error {
	tmpDir := s.tmpDir
	dirs := []string{tmpDir.base, tmpDir.app, tmpDir.lib}
	var mkTmpDirs = func(dirs []string) error {
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
				return err
			}
		}
		return nil
	}

	var rmTmpDirs = func(dirs []string) error {
		for _, dir := range dirs {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
		return nil
	}

	if err := rmTmpDirs(dirs); err != nil {
		return err
	}
	if reCreate {
		if err := mkTmpDirs(dirs); err != nil {
			return err
		}
	}
	return nil
}

// processTarName will trim the prefix of image name like example.io/library/myapp:v1
// after processed, the name will be myapp_v1_suffix
// mind: suffix here should not contain path separator
func (info imageInfo) processTarName(suffix string) string {
	originNames := strings.Split(info.name, string(os.PathSeparator))
	originTags := strings.Split(info.tag, string(os.PathSeparator))
	// get the last element of the list, which mast be the right name without prefix
	name := originNames[len(originNames)-1]
	tag := originTags[len(originTags)-1]

	return fmt.Sprintf("%s_%s%s", name, tag, suffix)
}

func (info *imageInfo) processBaseImg(sep *separatorSave, baseImagesMap map[string]string, tarball *tarballInfo) error {
	// process base
	tarball.BaseImageName = sep.base
	for _, layerID := range info.layers.base {
		tarball.BaseLayers = append(tarball.BaseLayers, layerID)
		if baseImg, ok := baseImagesMap[layerID]; !ok {
			srcLayerPath := filepath.Join(sep.tmpDir.untar, layerID)
			destLayerPath := filepath.Join(sep.tmpDir.base, layerID)
			if err := os.Rename(srcLayerPath, destLayerPath); err != nil {
				return err
			}
			baseTarName := info.processTarName(baseTarNameSuffix)
			baseTarName = sep.rename(baseTarName)
			baseTarPath := filepath.Join(sep.dest, baseTarName)
			if err := util.PackFiles(sep.tmpDir.base, baseTarPath, archive.Gzip, true); err != nil {
				return err
			}
			baseImagesMap[layerID] = baseTarPath
			tarball.BaseTarName = baseTarName
			digest, err := util.SHA256Sum(baseTarPath)
			if err != nil {
				return errors.Wrapf(err, "check sum for new base image %s failed", baseTarName)
			}
			tarball.BaseHash = digest
		} else {
			tarball.BaseTarName = filepath.Base(baseImg)
			digest, err := util.SHA256Sum(baseImg)
			if err != nil {
				return errors.Wrapf(err, "check sum for reuse base image %s failed", baseImg)
			}
			tarball.BaseHash = digest
		}
	}

	return nil
}

func (info *imageInfo) processLibImg(sep *separatorSave, libImagesMap map[string]string, tarball *tarballInfo) error {
	// process lib
	if info.layers.lib == nil {
		return nil
	}

	tarball.LibImageName = sep.lib
	for _, layerID := range info.layers.lib {
		tarball.LibLayers = append(tarball.LibLayers, layerID)
		if libImg, ok := libImagesMap[layerID]; !ok {
			srcLayerPath := filepath.Join(sep.tmpDir.untar, layerID)
			destLayerPath := filepath.Join(sep.tmpDir.lib, layerID)
			if err := os.Rename(srcLayerPath, destLayerPath); err != nil {
				return err
			}
			libTarName := info.processTarName(libTarNameSuffix)
			libTarName = sep.rename(libTarName)
			libTarPath := filepath.Join(sep.dest, libTarName)
			if err := util.PackFiles(sep.tmpDir.lib, libTarPath, archive.Gzip, true); err != nil {
				return err
			}
			libImagesMap[layerID] = libTarPath
			tarball.LibTarName = libTarName
			digest, err := util.SHA256Sum(libTarPath)
			if err != nil {
				return errors.Wrapf(err, "check sum for lib image %s failed", sep.lib)
			}
			tarball.LibHash = digest
		} else {
			tarball.LibTarName = filepath.Base(libImg)
			digest, err := util.SHA256Sum(libImg)
			if err != nil {
				return errors.Wrapf(err, "check sum for lib image %s failed", sep.lib)
			}
			tarball.LibHash = digest
		}
	}

	return nil
}

func (info *imageInfo) processAppImg(sep *separatorSave, appImagesMap map[string]string, tarball *tarballInfo) error {
	// process app
	appTarName := info.processTarName(appTarNameSuffix)
	appTarName = sep.rename(appTarName)
	appTarPath := filepath.Join(sep.dest, appTarName)
	for _, layerID := range info.layers.app {
		srcLayerPath := filepath.Join(sep.tmpDir.untar, layerID)
		destLayerPath := filepath.Join(sep.tmpDir.app, layerID)
		if err := os.Rename(srcLayerPath, destLayerPath); err != nil {
			if appImg, ok := appImagesMap[layerID]; ok {
				return errors.Errorf("lib layers %s already saved in %s for image %s",
					layerID, appImg, info.nameTag)
			}
		}
		appImagesMap[layerID] = appTarPath
		tarball.AppLayers = append(tarball.AppLayers, layerID)
	}
	// create config file
	if err := info.createManifestFile(sep); err != nil {
		return err
	}
	if err := info.createRepositoriesFile(sep); err != nil {
		return err
	}

	srcConfigPath := filepath.Join(sep.tmpDir.untar, info.config)
	destConfigPath := filepath.Join(sep.tmpDir.app, info.config)
	if err := os.Rename(srcConfigPath, destConfigPath); err != nil {
		return err
	}

	if err := util.PackFiles(sep.tmpDir.app, appTarPath, archive.Gzip, true); err != nil {
		return err
	}
	tarball.AppTarName = appTarName
	digest, err := util.SHA256Sum(appTarPath)
	if err != nil {
		return errors.Wrapf(err, "check sum for app image %s failed", info.nameTag)
	}
	tarball.AppHash = digest

	return nil
}

func (info imageInfo) createRepositoriesFile(sep *separatorSave) error {
	// create repositories
	type repoItem map[string]string
	repo := make(map[string]repoItem, 1)
	item := make(repoItem, 1)
	if _, ok := item[info.tag]; !ok {
		item[info.tag] = info.topLayer
	}
	repo[info.name] = item
	buf, err := json.Marshal(repo)
	if err != nil {
		return err
	}
	repositoryFile := filepath.Join(sep.tmpDir.app, repositoriesFile)
	if err := ioutils.AtomicWriteFile(repositoryFile, buf, constant.DefaultRootFileMode); err != nil {
		return err
	}
	return nil
}

func (info imageInfo) createManifestFile(sep *separatorSave) error {
	// create manifest.json
	var s = imageManifest{
		Config:   info.config,
		Layers:   info.layers.all,
		RepoTags: info.repoTags,
	}
	var m []imageManifest
	m = append(m, s)
	buf, err := json.Marshal(&m)
	if err != nil {
		return err
	}
	data := filepath.Join(sep.tmpDir.app, manifestDataFile)
	if err := ioutils.AtomicWriteFile(data, buf, constant.DefaultRootFileMode); err != nil {
		return err
	}
	return nil
}

func getLayersID(layer []string) []string {
	var after = make([]string, len(layer))
	for i, v := range layer {
		after[i] = strings.Split(v, "/")[0]
	}
	return after
}

func (s *separatorSave) constructSingleImgInfo(mani imageManifest, store *store.Store) (imageInfo, error) {
	var libLayers, appLayers []string
	imageRepoFields := strings.Split(mani.RepoTags[0], ":")
	imageLayers := getLayersID(mani.Layers)

	libs, bases, err := s.checkLayersHash(mani.HashMap, store)
	if err != nil {
		return imageInfo{}, errors.Wrap(err, "compare layers failed")
	}
	baseLayers := imageLayers[0:len(bases)]
	if len(libs) != 0 {
		libLayers = imageLayers[len(bases):len(libs)]
		appLayers = imageLayers[len(libs):]
	} else {
		libLayers = nil
		appLayers = imageLayers[len(bases):]
	}

	return imageInfo{
		config:   mani.Config,
		repoTags: mani.RepoTags,
		nameTag:  mani.RepoTags[0],
		name:     strings.Join(imageRepoFields[0:len(imageRepoFields)-1], ":"),
		tag:      imageRepoFields[len(imageRepoFields)-1],
		layers:   layer{app: appLayers, lib: libLayers, base: baseLayers, all: mani.Layers},
		topLayer: imageLayers[len(imageLayers)-1],
	}, nil
}

func (s *separatorSave) checkLayersHash(layerHashMap map[string]string, store *store.Store) ([]string, []string, error) {
	libHash, err := getLayerHashFromStorage(store, s.lib)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get lib image %s layers failed", s.lib)
	}
	baseHash, err := getLayerHashFromStorage(store, s.base)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get base image %s layers failed", s.base)
	}
	if len(libHash) >= len(layerHashMap) || len(baseHash) >= len(layerHashMap) {
		return nil, nil, errors.Errorf("number of base or lib layers is equal or greater than saved app layers")
	}

	for _, l := range libHash {
		if _, ok := layerHashMap[l]; !ok {
			return nil, nil, errors.Errorf("dismatch checksum for lib image %s", s.lib)
		}
	}
	for _, b := range baseHash {
		if _, ok := layerHashMap[b]; !ok {
			return nil, nil, errors.Errorf("dismatch checksum for base image %s", s.base)
		}
	}

	return libHash, baseHash, nil
}

func (s *separatorSave) constructImageInfos(manifest []imageManifest, store *store.Store) (map[string]imageInfo, error) {
	s.log.Info("Constructing image info")

	var imgInfos = make(map[string]imageInfo, 1)
	for _, mani := range manifest {
		imgInfo, err := s.constructSingleImgInfo(mani, store)
		if err != nil {
			s.log.Errorf("Constructing image info failed: %v", err)
			return nil, errors.Wrap(err, "construct image info failed")
		}
		if _, ok := imgInfos[imgInfo.nameTag]; !ok {
			imgInfos[imgInfo.nameTag] = imgInfo
		}
	}
	return imgInfos, nil
}

func (s *separatorSave) rename(name string) string {
	if len(s.renameData) != 0 {
		s.log.Info("Renaming image tarballs")
		for _, item := range s.renameData {
			if item.Name == name {
				return item.Rename
			}
		}
	}
	return name
}
