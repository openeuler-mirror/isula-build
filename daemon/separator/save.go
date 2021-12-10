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
// Create: 2021-12-09
// Description: This file is handling "save" part for image separator at server side

package separator

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

// Saver the main instance for saving separated images
type Saver struct {
	log        *logrus.Entry
	renameData []renames
	tmpDir     imageTmpDir
	base       string
	lib        string
	dest       string
	renameFile string
	enabled    bool
}

type renames struct {
	OriName string `json:"name"`
	NewName string `json:"rename"`
}

type imageTmpDir struct {
	app   string
	base  string
	lib   string
	untar string
	root  string
}

type imageLayersMap map[string]string

// GetSepSaveOptions returns Save instance from SaveRequest
func GetSepSaveOptions(req *pb.SaveRequest, logEntry *logrus.Entry, dataRoot string) (Saver, string) {
	var tmpRoot = filepath.Join(dataRoot, filepath.Join(constant.DataRootTmpDirPrefix, req.GetSaveID()))
	var sep = Saver{
		base:       req.GetSep().GetBase(),
		lib:        req.GetSep().GetLib(),
		dest:       req.GetSep().GetDest(),
		log:        logEntry,
		enabled:    req.GetSep().GetEnabled(),
		renameFile: req.GetSep().GetRename(),

		tmpDir: imageTmpDir{
			root:  tmpRoot,
			untar: filepath.Join(tmpRoot, untarTempDirName),
			base:  filepath.Join(tmpRoot, baseUntarTempDirName),
			lib:   filepath.Join(tmpRoot, libUntarTempDirName),
			app:   filepath.Join(tmpRoot, appUntarTempDirName),
		},
	}

	return sep, filepath.Join(sep.tmpDir.untar, unionTarName)
}

// SeparateImage the main method of Saver, tries to separated the listed images to pieces
func (s *Saver) SeparateImage(localStore *store.Store, oriImgList []string, outputPath string) (err error) {
	s.log.Infof("Start saving separated images %v", oriImgList)

	if err = os.MkdirAll(s.dest, constant.DefaultRootDirMode); err != nil {
		return err
	}

	defer func() {
		if tErr := os.RemoveAll(s.tmpDir.root); tErr != nil && !os.IsNotExist(tErr) {
			s.log.Warnf("Removing save tmp directory %q failed: %v", s.tmpDir.root, tErr)
		}
		if err != nil {
			if rErr := os.RemoveAll(s.dest); rErr != nil && !os.IsNotExist(rErr) {
				s.log.Warnf("Removing save dest directory %q failed: %v", s.dest, rErr)
			}
		}
	}()
	if err = util.UnpackFile(outputPath, s.tmpDir.untar, archive.Gzip, true); err != nil {
		return errors.Wrapf(err, "unpack %q failed", outputPath)
	}
	manifest, aErr := s.adjustLayers()
	if aErr != nil {
		return errors.Wrap(aErr, "adjust layers failed")
	}

	imgInfos, cErr := s.constructImageInfos(manifest, localStore)
	if cErr != nil {
		return errors.Wrap(cErr, "process image infos failed")
	}

	if err = s.processImageLayers(imgInfos); err != nil {
		return err
	}

	return nil
}

func (s *Saver) getLayerHashFromStorage(store *store.Store, name string) ([]string, error) {
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
func (s *Saver) constructLayerMap() (map[string]string, error) {
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

func (s *Saver) adjustLayers() ([]imageManifest, error) {
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

func (s *Saver) processImageLayers(imgInfos map[string]imageInfo) error {
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
		if err := s.clearTempDirs(); err != nil {
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

func (s *Saver) clearTempDirs() error {
	dirs := []string{s.tmpDir.base, s.tmpDir.app, s.tmpDir.lib}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
			return err
		}
	}
	return nil
}

// ImageNames returns the images names of Saver
func (s *Saver) ImageNames() []string {
	var names = make([]string, 0, 2)
	if !s.enabled {
		return []string{}
	}
	if len(s.base) != 0 {
		names = append(names, s.base)
	}
	if len(s.lib) != 0 {
		names = append(names, s.lib)
	}
	return names
}

func (s *Saver) constructSingleImgInfo(mani imageManifest, store *store.Store) (imageInfo, error) {
	var libLayers, appLayers []string
	// image name should not be empty here
	if len(mani.RepoTags) == 0 {
		return imageInfo{}, errors.New("image name and tag is empty")
	}
	// if there is more than one repoTag, will use first one as image name
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

func (s *Saver) checkLayersHash(layerHashMap map[string]string, store *store.Store) ([]string, []string, error) {
	libHash, err := s.getLayerHashFromStorage(store, s.lib)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get lib image %s layers failed", s.lib)
	}
	baseHash, err := s.getLayerHashFromStorage(store, s.base)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get base image %s layers failed", s.base)
	}
	if len(baseHash) > 1 {
		return nil, nil, errors.Errorf("number of base layers %d more than one", len(baseHash))
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

func (s *Saver) constructImageInfos(manifest []imageManifest, store *store.Store) (map[string]imageInfo, error) {
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

// LoadRenameFile Saver tries to load the specified rename json
func (s *Saver) LoadRenameFile() error {
	if len(s.renameFile) == 0 {
		return nil
	}

	var reName []renames
	if err := util.LoadJSONFile(s.renameFile, &reName); err != nil {
		return errors.Wrap(err, "check rename file failed")
	}
	s.renameData = reName
	return nil
}

func (s *Saver) getRename(name string) string {
	if len(s.renameData) == 0 {
		return name
	}

	for _, item := range s.renameData {
		if item.OriName == name {
			s.log.Infof("Renaming image tarballs for %s to %s\n", name, item.NewName)
			return item.NewName
		}
	}
	return name
}

// Enabled returns whether separated-image feature is enabled
func (s *Saver) Enabled() bool {
	return s.enabled
}
