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
// Description: This file is handling "load" part for image separator at server side

package separator

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/archive"
	filepath_securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/util"
)

type loadImageTmpDir struct {
	app  string
	base string
	lib  string
	root string
}

// Loader the main instance for loading separated images
type Loader struct {
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

// GetSepLoadOptions returns Loader instance from LoadRequest
func GetSepLoadOptions(req *pb.LoadRequest, logEntry *logrus.Entry, dataRoot string) (Loader, error) {
	var tmpRoot = filepath.Join(dataRoot, filepath.Join(constant.DataRootTmpDirPrefix, req.GetLoadID()))
	var sep = Loader{
		appName:  req.GetSep().GetApp(),
		basePath: req.GetSep().GetBase(),
		libPath:  req.GetSep().GetLib(),
		dir:      req.GetSep().GetDir(),
		log:      logEntry,
		tmpDir: loadImageTmpDir{
			root: tmpRoot,
			base: filepath.Join(tmpRoot, tmpBaseDirName),
			lib:  filepath.Join(tmpRoot, tmpLibDirName),
			app:  filepath.Join(tmpRoot, tmpAppDirName),
		},
		skipCheck: req.GetSep().GetSkipCheck(),
		enabled:   req.GetSep().GetEnabled(),
	}

	// check image name and add "latest" tag if not present
	_, appImgName, err := image.GetNamedTaggedReference(sep.appName)
	if err != nil {
		return Loader{}, err
	}
	if len(appImgName) == 0 {
		return Loader{}, errors.New("app image name should not be empty")
	}
	sep.appName = appImgName
	return sep, nil
}

// LoadSeparatedImage the main method of Loader, tries to load the separated images, and returns the path of the
// reconstructed image tarball for later handling
func (l *Loader) LoadSeparatedImage() (string, error) {
	l.log.Infof("Starting load separated image %s", l.appName)

	// load manifest file to get tarball info
	if err := l.getTarballInfo(); err != nil {
		return "", errors.Wrap(err, "failed to get tarball info")
	}
	if err := l.constructLayerPath(); err != nil {
		return "", err
	}
	// checksum for image tarballs
	if err := l.tarballCheckSum(); err != nil {
		return "", err
	}
	// process image tarballs and get final constructed image tarball
	return l.processTarballs()
}

func (l *Loader) getTarballInfo() error {
	manifest, err := filepath_securejoin.SecureJoin(l.dir, manifestFile)
	if err != nil {
		return errors.Wrap(err, "join manifest file path failed")
	}

	var t = make(map[string]tarballInfo, 1)
	if err = util.LoadJSONFile(manifest, &t); err != nil {
		return errors.Wrap(err, "load manifest file failed")
	}

	tarball, ok := t[l.appName]
	if !ok {
		return errors.Errorf("failed to find app image %s", l.appName)
	}
	if len(tarball.AppTarName) == 0 {
		return errors.Errorf("app image %s tarball can not be empty", tarball.AppTarName)
	}
	if len(tarball.BaseTarName) == 0 {
		return errors.Errorf("base image %s tarball can not be empty", tarball.BaseImageName)
	}
	l.info = tarball

	return nil
}

func (l *Loader) joinPath(path, tarName, str string, canBeEmpty bool) (string, error) {
	if len(path) != 0 {
		return path, nil
	}
	l.log.Infof("%s image path is empty, use path from manifest", str)
	return filepath_securejoin.SecureJoin(l.dir, tarName)
}

func (l *Loader) constructLayerPath() error {
	l.log.Infof("Construct image layer pathes for %s\n", l.appName)

	var err error
	if l.basePath, err = l.joinPath(l.basePath, l.info.BaseTarName, "Base", false); err != nil {
		return err
	}
	if l.libPath, err = l.joinPath(l.libPath, l.info.LibTarName, "Lib", true); err != nil {
		return err
	}
	if l.appPath, err = l.joinPath(l.appPath, l.info.AppTarName, "App", false); err != nil {
		return err
	}

	return nil
}

func (l *Loader) tarballCheckSum() error {
	if l.skipCheck {
		l.log.Info("Skip checksum for tarballs")
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
	checkList = append(checkList, checkInfo{path: l.basePath, hash: l.info.BaseHash, canBeEmpty: false, str: "base"})
	checkList = append(checkList, checkInfo{path: l.libPath, hash: l.info.LibHash, canBeEmpty: true, str: "lib"})
	checkList = append(checkList, checkInfo{path: l.appPath, hash: l.info.AppHash, canBeEmpty: false, str: "app"})
	for _, p := range checkList {
		if len(p.path) == 0 && !p.canBeEmpty {
			return errors.Errorf("%s image tarball path can not be empty", p.str)
		}
		if len(p.path) != 0 {
			if err := util.CheckSum(p.path, p.hash); err != nil {
				return errors.Wrapf(err, "check sum for file %q failed", p.path)
			}
		}
	}

	return nil
}

func (l *Loader) processTarballs() (string, error) {
	if err := l.unpackTarballs(); err != nil {
		return "", err
	}

	if err := l.reconstructImage(); err != nil {
		return "", err
	}

	// pack app image to tarball
	tarPath := filepath.Join(l.tmpDir.root, unionCompressedTarName)
	if err := util.PackFiles(l.tmpDir.base, tarPath, archive.Gzip, true); err != nil {
		return "", err
	}

	return tarPath, nil
}

func (l *Loader) unpackTarballs() error {
	if err := l.makeTempDir(); err != nil {
		return errors.Wrap(err, "failed to make temporary directories")
	}

	type unpackInfo struct{ path, dir, str string }
	unpackLen := 3
	var unpackList = make([]unpackInfo, 0, unpackLen)
	unpackList = append(unpackList, unpackInfo{path: l.basePath, dir: l.tmpDir.base, str: "base"})
	unpackList = append(unpackList, unpackInfo{path: l.libPath, dir: l.tmpDir.lib, str: "lib"})
	unpackList = append(unpackList, unpackInfo{path: l.appPath, dir: l.tmpDir.app, str: "app"})

	for _, p := range unpackList {
		if len(p.path) != 0 {
			if err := util.UnpackFile(p.path, p.dir, archive.Gzip, false); err != nil {
				return errors.Wrapf(err, "unpack %s image tarball %q failed", p.str, p.path)
			}
		}
	}

	return nil
}

func (l *Loader) reconstructImage() error {
	files, err := ioutil.ReadDir(l.tmpDir.app)
	if err != nil {
		return err
	}

	for _, f := range files {
		src := filepath.Join(l.tmpDir.app, f.Name())
		dest := filepath.Join(l.tmpDir.base, f.Name())
		if err := os.Rename(src, dest); err != nil {
			return errors.Wrapf(err, "reconstruct app file %q failed", l.info.AppTarName)
		}
	}

	if len(l.libPath) != 0 {
		files, err := ioutil.ReadDir(l.tmpDir.lib)
		if err != nil {
			return err
		}

		for _, f := range files {
			src := filepath.Join(l.tmpDir.lib, f.Name())
			dest := filepath.Join(l.tmpDir.base, f.Name())
			if err := os.Rename(src, dest); err != nil {
				return errors.Wrapf(err, "reconstruct lib file %q failed", l.info.LibTarName)
			}
		}
	}

	return nil
}

func (l *Loader) makeTempDir() error {
	dirs := []string{l.tmpDir.root, l.tmpDir.app, l.tmpDir.base, l.tmpDir.lib}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, constant.DefaultRootDirMode); err != nil {
			return err
		}
	}

	return nil
}

// AppName returns the AppName of Loader
func (l *Loader) AppName() string {
	return l.appName
}

// TmpDirRoot returns the tmpDir.root of Loader
func (l *Loader) TmpDirRoot() string {
	return l.tmpDir.root
}

// Enabled returns whether separated-image feature is enabled
func (l *Loader) Enabled() bool {
	return l.enabled
}
