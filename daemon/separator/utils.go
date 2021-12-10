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
// Description: This file is utils for image separator

package separator

import "strings"

const (
	manifestDataFile = "manifest.json"
	manifestFile     = "manifest"
	repositoriesFile = "repositories"

	tmpBaseDirName       = "base"
	tmpAppDirName        = "app"
	tmpLibDirName        = "lib"
	baseUntarTempDirName = "base_images"
	appUntarTempDirName  = "app_images"
	libUntarTempDirName  = "lib_images"
	baseTarNameSuffix    = "_base_image.tar.gz"
	appTarNameSuffix     = "_app_image.tar.gz"
	libTarNameSuffix     = "_lib_image.tar.gz"

	unionTarName           = "all.tar"
	unionCompressedTarName = "all.tar.gz"

	untarTempDirName = "untar"
	layerTarName     = "layer.tar"
	tarSuffix        = ".tar"
)

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
	BaseLayer     string   `json:"baseLayer"`
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

func getLayersID(layer []string) []string {
	var after = make([]string, len(layer))
	for i, v := range layer {
		after[i] = strings.Split(v, "/")[0]
	}
	return after
}
