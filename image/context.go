// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zhongkai Lei
// Create: 2020-03-20
// Description: image context related functions

package image

import (
	"io"
	"os"
	"sync"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/util"
)

var (
	globalSystemContext types.SystemContext
	once                sync.Once
)

func init() {
	globalSystemContext = types.SystemContext{
		SignaturePolicyPath:         constant.SignaturePolicyPath,
		SystemRegistriesConfDirPath: constant.RegistryConfigPath,
		RegistriesDirPath:           constant.RegistryDirPath,
	}
}

func validateConfigFiles(configs []string) error {
	var (
		cfgInfo os.FileInfo
		err     error
	)
	for _, cfg := range configs {
		if err = util.CheckFileSize(cfg, constant.MaxFileSize); err != nil {
			return err
		}
		if cfgInfo, err = os.Stat(cfg); err != nil {
			return err
		}
		if cfgInfo.Size() == 0 {
			return errors.Errorf("config %q cannot be an empty file", cfg)
		}
	}

	return nil
}

// SetSystemContext set the values of globalSystemContext
func SetSystemContext(dataRoot string) {
	err := validateConfigFiles([]string{constant.SignaturePolicyPath, constant.RegistryConfigPath})
	if err != nil {
		logrus.Fatal(err)
	}

	once.Do(func() {
		globalSystemContext.SignaturePolicyPath = constant.SignaturePolicyPath
		globalSystemContext.SystemRegistriesConfPath = constant.RegistryConfigPath
		globalSystemContext.RegistriesDirPath = constant.RegistryDirPath
		globalSystemContext.BlobInfoCacheDir = dataRoot
		globalSystemContext.AuthFilePath = constant.AuthFilePath
	})
}

// GetSystemContext returns the COPY of globalSystemContext
func GetSystemContext() *types.SystemContext {
	sc := globalSystemContext

	return &sc
}

// NewImageCopyOptions returns a copy options for copy.Image call
func NewImageCopyOptions(reportWriter io.Writer) *cp.Options {
	return &cp.Options{
		ReportWriter:   reportWriter,
		SourceCtx:      GetSystemContext(),
		DestinationCtx: GetSystemContext(),
	}
}
