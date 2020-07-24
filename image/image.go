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
// Description: image related functions

// Package image includes image related functions
package image

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/containers/image/v5/signature"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/stringid"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	dockerfile "isula.org/isula-build/builder/dockerfile/parser"
	"isula.org/isula-build/pkg/docker"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

var (
	// DefaultRegistryConfigPath describes the config path of registries
	DefaultRegistryConfigPath = "/etc/isula-build/registries.toml"

	// DefaultSignaturePolicyPath describes the policy path
	DefaultSignaturePolicyPath = "/etc/isula-build/policy.json"

	// DefaultRegistryDirPath is the dir to store registry configs
	DefaultRegistryDirPath = "/etc/isula-build/registries.d"

	// DefaultBlobInfoCacheDirPath is the dir to cache blob info
	DefaultBlobInfoCacheDirPath = "/var/lib/isula-build"

	// DefaultAuthFile is authentication file used for registry connection
	DefaultAuthFile = "/etc/isula-build/auth.json"
)

// PrepareImageOptions describes the options required for preparing the image
type PrepareImageOptions struct {
	SystemContext *types.SystemContext
	Ctx           context.Context
	FromImage     string
	Store         store.Store
	Reporter      io.Writer
}

// ContainerDescribe describes the contents for container
type ContainerDescribe struct {
	ContainerName string
	ContainerID   string
	Mountpoint    string
}

// Describe describes the prepared image
type Describe struct {
	ContainerDesc *ContainerDescribe
	Image         types.Image
	ImageID       string
	TopLayID      string
}

type pullOption struct {
	sc       *types.SystemContext
	ctx      context.Context
	reporter io.Writer

	srcRef  types.ImageReference
	dstRef  types.ImageReference
	dstName string
}

func pullImage(opt pullOption) (types.ImageReference, error) {
	pLog := logrus.WithField(util.LogKeyBuildID, opt.ctx.Value(util.LogFieldKey(util.LogKeyBuildID)))
	policy, err := signature.DefaultPolicy(opt.sc)
	if err != nil {
		return nil, errors.Wrapf(err, "error obtaining default signature policy")
	}

	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating new signature policy context")
	}

	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			pLog.Debugf("Error destroying signature policy context: %v", err2)
		}
	}()

	cpOpt := &copy.Options{
		ReportWriter:   opt.reporter,
		SourceCtx:      opt.sc,
		DestinationCtx: GetSystemContext(),
	}
	pLog.Debugf("Copying %q to %q", transports.ImageName(opt.srcRef), opt.dstName)
	if _, err := copy.Image(opt.ctx, policyContext, opt.dstRef, opt.srcRef, cpOpt); err != nil {
		pLog.Debugf("Error copying src image [%q] to dest image [%q] err: %v", transports.ImageName(opt.srcRef), opt.dstName, err)
		return nil, err
	}
	return opt.dstRef, nil
}

func pullAndGetImageInfo(opt *PrepareImageOptions) (types.ImageReference, *storage.Image, error) {
	pLog := logrus.WithField(util.LogKeyBuildID, opt.Ctx.Value(util.LogFieldKey(util.LogKeyBuildID)))
	candidates, transport, err := resolveName(opt.FromImage, opt.SystemContext, opt.Store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing reference to image %q", opt.FromImage)
	}

	if transport == "" {
		// if the image can be obtained from the local storage by image id,
		// then only one image can be obtained.
		if len(candidates) != 1 {
			return nil, nil, errors.New("transport is empty and multi or no image be found")
		}
		img, err := opt.Store.Image(candidates[0])
		if err != nil {
			pLog.Errorf("Failed to find the image %q in local store: %v", candidates[0], err)
			return nil, nil, err
		}
		ref, err := is.Transport.ParseStoreReference(opt.Store, img.ID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get the ref in store by %q", candidates[0])
		}
		pLog.Infof("Get image from local store first search by %q", opt.FromImage)
		return ref, img, nil
	}

	// record the last pull error
	var errPull error
	const tagSeperator = ":"
	for _, strImage := range candidates {
		if transport != util.DefaultTransport {
			transport += tagSeperator
		}
		srcRef, err := alltransports.ParseImageName(transport + strImage)
		if err != nil {
			pLog.Debugf("Failed to parse the image %q: %v", transport+strImage, err)
			continue
		}

		destImage, err := getLocalImageNameFromRef(opt.Store, srcRef)
		if err != nil {
			pLog.Debugf("Failed to get local image name for %q: %v", transport+strImage, err)
			continue
		}

		destRef, err := is.Transport.ParseStoreReference(opt.Store, destImage)
		if err != nil {
			pLog.Debugf("Failed to parse store reference for %q: %v", destImage, err)
			continue
		}

		img, err := is.Transport.GetStoreImage(opt.Store, destRef)
		if err == nil {
			// find the unique image in local store by name or digest
			pLog.Infof("Get image from local store second search by %q", opt.FromImage)
			return destRef, img, nil
		}

		// can not find image in local store, pull from registry
		pulledRef, err := pullImage(pullOption{
			ctx:      opt.Ctx,
			reporter: opt.Reporter,
			sc:       opt.SystemContext,
			srcRef:   srcRef,
			dstRef:   destRef,
			dstName:  destImage,
		})
		if err != nil {
			pLog.Debugf("Failed to pull image %q: %v", transport+strImage, err)
			continue
		}
		pulledImg, err := is.Transport.GetStoreImage(opt.Store, pulledRef)
		if err != nil {
			errPull = err
			pLog.Infof("Failed to obtaining pulled image %q: %v", transports.ImageName(srcRef), err)
			continue
		}
		return pulledRef, pulledImg, nil
	}

	return nil, nil, errors.Errorf("failed to get the image in %#v: %v", candidates, errPull)
}

func instantiatingImage(ctx context.Context, sc *types.SystemContext, ref types.ImageReference) (types.Image, error) {
	imgSource, err := ref.NewImageSource(ctx, sc)
	if err != nil {
		return nil, errors.Wrapf(err, "instantiating image %q failed", transports.ImageName(ref))
	}
	defer func() {
		if cerr := imgSource.Close(); cerr != nil {
			logrus.Warningf("Closing imgSource failed: %v", cerr)
		}
	}()
	byteManifest, mType, err := imgSource.GetManifest(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "loading image %q manifest failed", transports.ImageName(ref))
	}

	var (
		instanceDigest *digest.Digest
		list           manifest.List
		instance       digest.Digest
	)
	if manifest.MIMETypeIsMultiImage(mType) {
		list, err = manifest.ListFromBlob(byteManifest, mType)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing image %q manifest as list failed", transports.ImageName(ref))
		}
		instance, err = list.ChooseInstance(sc)
		if err != nil {
			return nil, errors.Wrapf(err, "finding the image in manifest list %q failed", transports.ImageName(ref))
		}
		instanceDigest = &instance
	}
	baseImg, err := image.FromUnparsedImage(ctx, sc, image.UnparsedInstance(imgSource, instanceDigest))
	if err != nil {
		return nil, errors.Wrapf(err, "instantiating image %q with instance %q failed", transports.ImageName(ref), instanceDigest)
	}

	return baseImg, nil
}

func getLocalImageNameFromRef(store storage.Store, srcRef types.ImageReference) (string, error) {
	if srcRef == nil {
		return "", errors.Errorf("reference to image is empty")
	}

	if srcRef.Transport().Name() == "docker-archive" {
		return stringid.GenerateRandomID() + ":" + stringid.GenerateRandomID(), nil
	}

	if srcRef.Transport().Name() != "docker" {
		return "", errors.Errorf("the %s transport is not supported yet", srcRef.Transport().Name())
	}

	var name string
	ref := srcRef.DockerReference()
	if ref == nil {
		return "", errors.New("get the docker reference associated with source reference failed")
	}

	if named, ok := ref.(reference.Named); ok {
		name = named.Name()
		if tag, ok := ref.(reference.NamedTagged); ok {
			name = name + ":" + tag.Tag()
		}
		if dig, ok := ref.(reference.Canonical); ok {
			name = name + "@" + dig.Digest().String()
		}
	}
	if _, err := is.Transport.ParseStoreReference(store, name); err != nil {
		return "", errors.Wrapf(err, "parsing image name %q failed", name)
	}

	return name, nil
}

func createScratchV2Image() *docker.Image {
	return &docker.Image{
		V1Image: docker.V1Image{
			ContainerConfig: docker.Config{},
			Config: &docker.Config{
				ExposedPorts: make(docker.PortSet),
				Env:          make([]string, 0, 0),
				Cmd:          make([]string, 0, 0),
				Healthcheck:  &docker.HealthConfig{},
				Volumes:      make(map[string]struct{}),
				Entrypoint:   make([]string, 0, 0),
				OnBuild:      make([]string, 0, 0),
				Labels:       make(map[string]string),
				StopTimeout:  nil,
				Shell:        make([]string, 0, 0),
			},
		},
		RootFS:  &docker.RootFS{},
		History: make([]docker.History, 0, 0),
	}
}

func createImageV2Image(ctx context.Context, fromImage types.Image, targetMIMEType string) (*docker.Image, error) {
	imageName := transports.ImageName(fromImage.Reference())
	_, imageMIMEType, err := fromImage.Manifest(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get MIME type from %q", imageName)
	}
	if targetMIMEType != imageMIMEType {
		updatedImg, err2 := fromImage.UpdatedImage(ctx, types.ManifestUpdateOptions{
			ManifestMIMEType: targetMIMEType,
		})
		if err2 != nil {
			return nil, errors.Wrapf(err, "failed to convert image %q", imageName)
		}
		fromImage = updatedImg
	}

	config, err := fromImage.ConfigBlob(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get config from %q", transports.ImageName(fromImage.Reference()))
	}

	var imgSpec docker.Image
	if err := json.Unmarshal(config, &imgSpec); err != nil {
		return nil, errors.Wrapf(err, "failed to parse config into %s", manifest.DockerV2Schema2MediaType)
	}

	return &imgSpec, nil
}

// UpdateV2Image update the image info depending on the current environment
func UpdateV2Image(docker *docker.Image) error {
	if docker == nil {
		return nil
	}

	if docker.Config != nil {
		docker.ContainerConfig = *docker.Config
	}
	docker.Config = &docker.ContainerConfig

	if docker.Created.IsZero() {
		docker.Created = time.Now().UTC()
	}

	if docker.OS == "" {
		docker.OS = runtime.GOOS
	}

	if docker.Architecture == "" {
		docker.Architecture = runtime.GOARCH
	}

	if docker.Architecture != runtime.GOARCH {
		// NOTE:cross-architecture build is not supported currently
		return errors.Errorf("the architecture does not match, have %q want %q", docker.Architecture, runtime.GOARCH)
	}

	if docker.Config.Hostname == "" {
		docker.Config.Hostname = "isula"
	}

	return nil
}

// ResolveFromImage pull the FROM image and instantiate it
func ResolveFromImage(opt *PrepareImageOptions) (types.Image, *storage.Image, error) {
	ref, si, err := pullAndGetImageInfo(opt)
	if err != nil {
		return nil, nil, err
	}

	img, err := instantiatingImage(opt.Ctx, opt.SystemContext, ref)
	if err != nil {
		return nil, nil, err
	}

	return img, si, nil
}

// GetRWLayerByImageID get the RW layer by image ID
func GetRWLayerByImageID(imgID string, store store.Store) (*ContainerDescribe, error) {
	var (
		container     *storage.Container
		err           error
		containerName string
	)

	for {
		randNum, rerr := util.GenerateCryptoNum(constant.DefaultIDLen)
		if rerr != nil {
			return nil, rerr
		}
		containerName = fmt.Sprintf("isula-build-%s", randNum)
		container, err = store.CreateContainer("", []string{containerName}, imgID, "", "", nil)
		if err == nil {
			break
		}
		if errors.Cause(err) != storage.ErrDuplicateName {
			return nil, errors.Wrapf(err, "error creating container")
		}
	}
	defer func() {
		if err != nil {
			if errRollBack := store.DeleteContainer(container.ID); errRollBack != nil {
				logrus.Warnf("Failed to deleting container %q in rollback: %v", container.ID, errRollBack)
			}
		}
	}()

	mountpoint, err := store.Mount(container.ID, "")
	if err != nil {
		return nil, errors.Wrapf(err, "error mounting build container %q", container.ID)
	}

	return &ContainerDescribe{
		ContainerName: containerName,
		ContainerID:   container.ID,
		Mountpoint:    mountpoint,
	}, nil
}

// GenerateFromImageSpec generate the image spec
func GenerateFromImageSpec(ctx context.Context, fromImage types.Image, targetMIMEType string) (*docker.Image, error) {
	var (
		docker *docker.Image
		err    error
	)

	if fromImage == nil {
		docker = createScratchV2Image()
	} else if docker, err = createImageV2Image(ctx, fromImage, targetMIMEType); err != nil {
		return nil, err
	}

	if err = UpdateV2Image(docker); err != nil {
		return nil, err
	}

	return docker, nil
}

// ResolveImageName resolves the params of image name in FROM command
// The image name format can be <name> or <name>:<tag> or <name>@<digest>
// and it can consists with params such as ${module}_${feature}_${platform}:${version}
func ResolveImageName(s string, resolveArg func(string) string) (string, error) {
	// check special case "\$", so we can better resolve param later
	newStr := strings.TrimSpace(s)
	if strings.Contains(newStr, "\\$") {
		return "", errors.Errorf("image name [%q] is invalid", s)
	}

	// convert name
	newStr, err := dockerfile.ResolveParam(newStr, true, resolveArg)
	if err != nil {
		return "", err
	}
	logrus.Infof("Input image name is %q, resolved to %q", s, newStr)

	// validate name
	if _, err := reference.Parse(newStr); err != nil {
		return "", err
	}
	return newStr, nil
}

// FindImage get the image from storage by image describe
func FindImage(store store.Store, image string) (types.ImageReference, *storage.Image, error) {
	names, _, err := resolveName(image, nil, store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing name %q", image)
	}

	var (
		ref types.ImageReference
		img *storage.Image
	)
	for _, name := range names {
		ref, err = is.Transport.ParseStoreReference(store, name)
		if err != nil {
			logrus.Debugf("Error parsing reference to image %q: %v", name, err)
			continue
		}
		img, err = is.Transport.GetStoreImage(store, ref)
		if err != nil {
			img2, err2 := store.Image(name)
			if err2 != nil {
				logrus.Debugf("Error locating image %q: %v", name, err2)
				continue
			}
			img = img2
		}
		break
	}
	if ref == nil || img == nil {
		return nil, nil, errors.Errorf("locating image %q with name: %v failed", image, names)
	}
	return ref, img, nil
}

func resolveName(name string, sc *types.SystemContext, store store.Store) ([]string, string, error) {
	// 1. check name valid
	if name == "" {
		return nil, "", nil
	}

	// 2. try to resolve image name as an image id and find it in store
	if imageID := tryResolveNameInStore(name, store); imageID != "" {
		return []string{imageID}, "", nil
	}

	// 3. try to resolve image name as a transport:destination format
	dest, transport := tryResolveNameWithTransport(name)
	if dest != "" && transport != "" {
		return []string{dest}, transport, nil
	}

	// 4. try to resolve image name as a docker reference
	// if error occurred in this step, there is no need to continue
	dest, transport, err := tryResolveNameWithDockerReference(name)
	if err != nil {
		return nil, "", err
	}
	if dest != "" && transport != "" {
		return []string{dest}, transport, nil
	}

	// 5. finally, try to resolve image name in registries
	candidates, transport := tryResolveNameInRegistries(name, sc)

	return candidates, transport, nil
}

func tryResolveNameInStore(name string, store store.Store) string {
	logrus.Infof("Try to resolve: %s in local storage with image id", name)
	img, err := store.Image(name)
	if err != nil || img == nil {
		return ""
	}

	if strings.HasPrefix(img.ID, name) {
		return img.ID
	}
	return ""
}

func tryResolveNameWithTransport(name string) (string, string) {
	logrus.Infof("Try to resolve name: %s with transport", name)
	splits := strings.SplitN(name, ":", 2)
	if len(splits) == 2 {
		if trans := transports.Get(splits[0]); trans != nil {
			switch trans.Name() {
			case "docker":
				// trim prefix if dest like docker://registry.example.com format
				dest := strings.TrimPrefix(splits[1], "//")
				return dest, trans.Name()
			case "docker-archive":
				dest := strings.TrimSpace(splits[1])
				return dest, trans.Name()
			}
		}
	}
	return "", ""
}

func tryResolveNameWithDockerReference(name string) (string, string, error) {
	logrus.Infof("Try to resolve name: %s with docker reference", name)
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return "", "", errors.Wrapf(err, "error parsing image name %q", name)
	}
	if named.String() == name {
		return name, util.DefaultTransport, nil
	}

	domain := reference.Domain(named)
	if pathPrefix, ok := util.DefaultRegistryPathPrefix[domain]; ok {
		repoPath := reference.Path(named)
		tag := ""
		if tagged, ok := named.(reference.Tagged); ok {
			tag = ":" + tagged.Tag()
		}
		digest := ""
		if digested, ok := named.(reference.Digested); ok {
			digest = "@" + digested.Digest().String()
		}
		defaultPrefix := pathPrefix + string(os.PathSeparator)
		if strings.HasPrefix(repoPath, defaultPrefix) && path.Join(domain, repoPath[len(defaultPrefix):])+tag+digest == name {
			return name, util.DefaultTransport, nil
		}
	}

	return "", "", nil
}

func tryResolveNameInRegistries(name string, sc *types.SystemContext) ([]string, string) {
	logrus.Infof("Try to resolve name: %s with in registries", name)
	var registries []string
	searchRegistries, err := sysregistriesv2.UnqualifiedSearchRegistries(sc)
	if err != nil {
		logrus.Debugf("Unable to read configured registries to complete %q: %v", name, err)
		searchRegistries = nil
	}
	for _, registry := range searchRegistries {
		reg, err := sysregistriesv2.FindRegistry(sc, registry)
		if err != nil {
			logrus.Debugf("Unable to read registry configuration for %#v: %v", registry, err)
			continue
		}
		if reg == nil || !reg.Blocked {
			registries = append(registries, registry)
		}
	}

	var candidates []string
	initRegistries := []string{"localhost"}
	for _, registry := range append(initRegistries, registries...) {
		if registry == "" {
			continue
		}
		middle := ""
		if prefix, ok := util.DefaultRegistryPathPrefix[registry]; ok && !strings.ContainsRune(name, '/') {
			middle = prefix
		}
		candidate := path.Join(registry, middle, name)
		candidates = append(candidates, candidate)
	}
	return candidates, util.DefaultTransport
}
