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
// Description: This file is handling image info for image separator

package separator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	"isula.org/isula-build/util"
)

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

// processTarName will trim the prefix of image name like example.io/library/myapp:v1
// after processed, the name will be myapp_v1_suffix
// mind: suffix here should not contain path separator
func (info *imageInfo) processTarName(suffix string) string {
	originNames := strings.Split(info.name, string(os.PathSeparator))
	originTags := strings.Split(info.tag, string(os.PathSeparator))
	// get the last element of the list, which mast be the right name without prefix
	name := originNames[len(originNames)-1]
	tag := originTags[len(originTags)-1]

	return fmt.Sprintf("%s_%s%s", name, tag, suffix)
}

func (info *imageInfo) processBaseImg(sep *Saver, baseImagesMap map[string]string, tarball *tarballInfo) error {
	// process base
	tarball.BaseImageName = sep.base
	if len(info.layers.base) != 0 {
		sep.log.Infof("Base image %s has %d layers", sep.base, len(info.layers.base))
		tarball.BaseLayer = info.layers.base[0]
	}
	for _, layerID := range info.layers.base {
		if baseImg, ok := baseImagesMap[layerID]; !ok {
			srcLayerPath := filepath.Join(sep.tmpDir.untar, layerID)
			destLayerPath := filepath.Join(sep.tmpDir.base, layerID)
			if err := os.Rename(srcLayerPath, destLayerPath); err != nil {
				return err
			}
			baseTarName := info.processTarName(baseTarNameSuffix)
			baseTarName = sep.getRename(baseTarName)
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

func (info *imageInfo) processLibImg(sep *Saver, libImagesMap map[string]string, tarball *tarballInfo) error {
	// process lib
	if info.layers.lib == nil {
		return nil
	}

	tarball.LibImageName = sep.lib
	sep.log.Infof("Lib image %s has %d layers", sep.lib, len(info.layers.lib))
	for _, layerID := range info.layers.lib {
		tarball.LibLayers = append(tarball.LibLayers, layerID)
		if libImg, ok := libImagesMap[layerID]; !ok {
			srcLayerPath := filepath.Join(sep.tmpDir.untar, layerID)
			destLayerPath := filepath.Join(sep.tmpDir.lib, layerID)
			if err := os.Rename(srcLayerPath, destLayerPath); err != nil {
				return err
			}
			libTarName := info.processTarName(libTarNameSuffix)
			libTarName = sep.getRename(libTarName)
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

func (info *imageInfo) processAppImg(sep *Saver, appImagesMap map[string]string, tarball *tarballInfo) error {
	// process app
	sep.log.Infof("App image %s has %d layers", info.nameTag, len(info.layers.app))
	appTarName := info.processTarName(appTarNameSuffix)
	appTarName = sep.getRename(appTarName)
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

func (info *imageInfo) createRepositoriesFile(sep *Saver) error {
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

// imageManifest return image's manifest info
type imageManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
	// Not shown in the json file
	HashMap map[string]string `json:"-"`
}

func (info *imageInfo) createManifestFile(sep *Saver) error {
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
