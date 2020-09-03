// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zhongkai Lei, Feiyu Yang
// Create: 2020-03-20
// Description: ADD and COPY command related functions

package dockerfile

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/fileutils"
	"github.com/containers/storage/pkg/idtools"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/image"
	"isula.org/isula-build/util"
)

type copyDetails map[string][]string

type copyOptions struct {
	// is ADD command
	isAdd bool
	// raw string of "--chown" value
	chown  string
	ignore []string
	// path get from request, or mountpoint of image or stage through "--from"
	contextDir string

	copyDetails copyDetails
}

type addOptions struct {
	// matcher matches the file which should be ignored
	matcher   *fileutils.PatternMatcher
	chownPair idtools.IDPair
	// extract is true and the tar file should be extracted
	extract bool
}

// resolveCopyDest gets the secure dest path and check validity
func resolveCopyDest(rawDest, workDir, mountpoint string) (string, error) {
	// in normal cases, the value of workdir obtained here must be an absolute path
	dest := util.MakeAbsolute(rawDest, workDir)

	secureMp, err := securejoin.SecureJoin("", mountpoint)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve symlinks for mountpoint %s", mountpoint)
	}
	secureDest, err := securejoin.SecureJoin(secureMp, dest)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve symlinks for destination %s", dest)
	}
	// ensure the destination path is in the mountpoint
	if !strings.HasPrefix(secureDest, secureMp) {
		return "", errors.Errorf("failed to resolve copy destination %s", rawDest)
	}
	if util.HasSlash(rawDest) && secureDest[len(secureDest)-1] != os.PathSeparator {
		secureDest += string(os.PathSeparator)
	}

	return secureDest, nil
}

// resolveCopySource gets the secureSource
func resolveCopySource(isAdd bool, rawSources []string, dest, contextDir string) (copyDetails, error) {
	details := make(copyDetails, len(rawSources))
	for _, src := range rawSources {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			if !isAdd {
				return nil, errors.Errorf("source can not be a URL for COPY")
			}
			if _, ok := details[dest]; !ok {
				details[dest] = []string{}
			}
			details[dest] = append(details[dest], src)
			continue
		}

		secureSrc, err := securejoin.SecureJoin(contextDir, src)
		if err != nil {
			return nil, errors.Wrapf(err, "%q is outside of the build context dir %q", src, secureSrc)
		}
		// if the destination is a folder, we may need to get the correct name
		// when the source file is a symlink
		if util.HasSlash(dest) {
			_, srcName := filepath.Split(src)
			_, SecureName := filepath.Split(secureSrc)
			if srcName != SecureName {
				newDest := filepath.Join(dest, srcName)
				if _, ok := details[newDest]; !ok {
					details[newDest] = []string{}
				}
				details[newDest] = append(details[newDest], secureSrc)
				continue
			}
		}
		if _, ok := details[dest]; !ok {
			details[dest] = []string{}
		}
		details[dest] = append(details[dest], secureSrc)
	}

	return details, nil
}

// getCopyContextDir gets the contextDir of from stage or image
func (c *cmdBuilder) getCopyContextDir(from string) (string, func(), error) {
	// the "from" parameter is a stage name
	if i, ok := c.stage.builder.stageAliasMap[from]; ok && i < c.stage.position {
		c.stage.builder.Logger().
			Debugf("Get context dir by stage name %q, context dir %q", from, c.stage.builder.stageBuilders[i].mountpoint)
		return c.stage.builder.stageBuilders[i].mountpoint, nil, nil
	}

	// try to consider of "from" as a stage index
	index, err := strconv.Atoi(from)
	if err == nil {
		if index >= 0 && index < c.stage.position {
			logrus.Debugf("Get context dir by stage index %q, context dir %q", index, c.stage.builder.stageBuilders[index].mountpoint)
			return c.stage.builder.stageBuilders[index].mountpoint, nil, nil
		}
	}

	// update cert path in case it is different between FROM and --from
	server, err := util.ParseServer(from)
	if err != nil {
		return "", nil, err
	}
	c.stage.buildOpt.systemContext.DockerCertPath = filepath.Join(constant.DefaultCertRoot, server)

	// "from" is neither name nor index of stage, consider that "from" is image description
	imgDesc, err := prepareImage(&image.PrepareImageOptions{
		Ctx:           c.ctx,
		FromImage:     from,
		SystemContext: c.stage.buildOpt.systemContext,
		Store:         c.stage.localStore,
		Reporter:      c.stage.builder.cliLog,
	})
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		if cerr := c.stage.localStore.CleanContainer(imgDesc.ContainerDesc.ContainerID); cerr != nil {
			logrus.Warnf("Clean layer[%s] for COPY from[%s] failed: %v", imgDesc.ContainerDesc.ContainerID, from, cerr)
		}
	}

	return imgDesc.ContainerDesc.Mountpoint, cleanup, nil
}

func (c *cmdBuilder) doCopy(opt *copyOptions) error {
	c.stage.builder.Logger().Debugf("copyOptions is %#v", opt)
	matcher, err := util.GetIgnorePatternMatcher(opt.ignore, opt.contextDir, filepath.Dir(c.stage.mountpoint))
	if err != nil {
		return err
	}

	chownPair, err := util.GetChownOptions(opt.chown, c.stage.mountpoint)
	if err != nil {
		return err
	}

	addOption := &addOptions{
		matcher:   matcher,
		chownPair: chownPair,
		extract:   opt.isAdd,
	}

	for dest, srcs := range opt.copyDetails {
		for _, src := range srcs {
			if err = c.add(src, dest, addOption); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *cmdBuilder) executeAddAndCopy(isAdd bool) error {
	allowFlags := map[string]bool{"chown": true}
	if !isAdd {
		allowFlags["from"] = true
	}

	// 1. get "--chown" , "--from" and args from c.line
	flags, args := getFlagsAndArgs(c.line, allowFlags)
	if len(args) < 2 {
		return errors.Errorf("the COPY/ADD args must contain at least src and dest")
	}

	// 2. get trustworthy and absolute destination
	dest := args[len(args)-1]
	// if there are multiple sources, dest must be a directory.
	if len(args) > 2 && !util.HasSlash(dest) {
		return errors.Errorf("%q is not a directory", dest)
	}
	finalDest, err := resolveCopyDest(dest, c.stage.docker.Config.WorkingDir, c.stage.mountpoint)
	if err != nil {
		return err
	}

	// 3. get context dir
	contextDir := c.stage.builder.buildOpts.ContextDir
	if from, ok := flags["from"]; ok {
		var cleanup func()
		if contextDir, cleanup, err = c.getCopyContextDir(from); err != nil {
			return err
		}
		defer func() {
			if cleanup != nil {
				cleanup()
			}
		}()
	}

	// 4. get all of secure sources
	details, err := resolveCopySource(isAdd, args[:len(args)-1], finalDest, contextDir)
	if err != nil {
		return err
	}

	var chown string
	if flag, ok := flags["chown"]; ok {
		chown = flag
	}

	// 5. do copy
	copyOpt := &copyOptions{
		isAdd:       isAdd,
		chown:       chown,
		ignore:      c.stage.builder.ignores,
		contextDir:  contextDir,
		copyDetails: details,
	}

	return c.doCopy(copyOpt)
}

func addDirectory(realSrc, dest string, opt *addOptions) error {
	if err := idtools.MkdirAllAndChownNew(dest, constant.DefaultSharedDirMode, opt.chownPair); err != nil {
		return errors.Wrapf(err, "error creating directory %q", dest)
	}

	logrus.Debugf("Copying directory from %q to %q", realSrc, dest)
	return filepath.Walk(realSrc, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if matched, merr := util.IsMatched(opt.matcher, path); merr != nil || matched {
			if info.IsDir() {
				merr = filepath.SkipDir
			}
			return merr
		}

		relativePath, rerr := filepath.Rel(realSrc, path)
		if rerr != nil {
			return rerr
		}
		destPath := filepath.Join(dest, relativePath)

		if info.IsDir() {
			// if the dest path is a file, remove it first
			fi, err := os.Lstat(destPath)
			if err == nil && !fi.IsDir() {
				if err := os.Remove(destPath); err != nil {
					return err
				}
			}
			return idtools.MkdirAllAndChownNew(destPath, info.Mode(), opt.chownPair)
		}

		if util.IsSymbolFile(path) {
			return util.CopySymbolFile(path, destPath, opt.chownPair)
		}

		return util.CopyFile(path, destPath, opt.chownPair)
	})
}

// addFile adds a single file, if extract is true and src is a archive file,
// extract it into dest, or treat it as a normal file.
func addFile(realSrc, globFile, dest string, opt *addOptions) error {
	if opt.matcher != nil {
		if matched, err := util.IsMatched(opt.matcher, realSrc); matched || err != nil {
			return err
		}
	}

	if !opt.extract || !archive.IsArchivePath(realSrc) {
		if strings.HasSuffix(dest, string(os.PathSeparator)) || util.IsDirectory(dest) {
			dest = filepath.Join(dest, filepath.Base(globFile))
		}

		logrus.Debugf("Copying single file from %q to %q", realSrc, dest)
		return util.CopyFile(realSrc, dest, opt.chownPair)
	}

	// The src is an archive file and extract is true,so extract it
	logrus.Debugf("Extracting from %q to %q", realSrc, dest)
	extractArchive := chrootarchive.UntarPathAndChown(nil, nil, nil, nil)
	return extractArchive(realSrc, dest)
}

func (c *cmdBuilder) add(src, dest string, opt *addOptions) error {
	// the src is URL
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return errors.New("URL is not support yet")
	}

	globFiles, err := filepath.Glob(src)
	if err != nil {
		return errors.Wrapf(err, "failed to get the glob by %q", src)
	}
	if len(globFiles) == 0 {
		return errors.Errorf("found no file that matches %q", src)
	}

	for _, globFile := range globFiles {
		realSrc, err := filepath.EvalSymlinks(globFile)
		if err != nil {
			return err
		}
		realSrcFileInfo, err := os.Stat(realSrc)
		if err != nil {
			return err
		}

		// if it is directory,walk and continue
		if realSrcFileInfo.IsDir() {
			if err = addDirectory(realSrc, dest, opt); err != nil {
				return err
			}
			continue
		}

		// realSrc is a single file
		if err = addFile(realSrc, globFile, dest, opt); err != nil {
			return err
		}
	}

	return nil
}
