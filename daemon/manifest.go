// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Danni Xia
// Create: 2020-12-01
// Description: This file is used for manifest command.

package daemon

import (
	"context"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	isulamanifest "isula.org/isula-build/pkg/manifest"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

// ManifestCreate creates manifest list
func (b *Backend) ManifestCreate(ctx context.Context, req *pb.ManifestCreateRequest) (*pb.ManifestCreateResponse, error) {
	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
		"Manifest":     req.GetManifests(),
	}).Info("ManifestCreateRequest received")

	if !b.daemon.opts.Experimental {
		logrus.WithField(util.LogKeySessionID, req.GetManifestList()).Error("Please enable experimental to use manifest feature")
		return &pb.ManifestCreateResponse{}, errors.New("please enable experimental to use manifest feature")
	}

	manifestName := req.GetManifestList()
	manifests := req.GetManifests()

	list := isulamanifest.NewManifestList()

	for _, imageSpec := range manifests {
		// add image to list
		if _, err := list.AddImage(ctx, b.daemon.localStore, imageSpec); err != nil {
			logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Add image to list err: %v", err)
			return &pb.ManifestCreateResponse{}, err
		}
	}

	// expand list name
	_, imageName, err := dockerfile.CheckAndExpandTag(manifestName)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Check and expand list name err: %v", err)
		return &pb.ManifestCreateResponse{}, err
	}
	// save list to image
	imageID, err := list.SaveListToImage(b.daemon.localStore, "", imageName)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Save list to image err: %v", err)
	}

	return &pb.ManifestCreateResponse{
		ImageID: imageID,
	}, err
}

// ManifestAnnotate modifies and updates manifest list
func (b *Backend) ManifestAnnotate(ctx context.Context, req *pb.ManifestAnnotateRequest) (*gogotypes.Empty, error) {
	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
		"Manifest":     req.GetManifest(),
	}).Info("ManifestAnnotateRequest received")

	var emptyResp = &gogotypes.Empty{}

	if !b.daemon.opts.Experimental {
		logrus.WithField(util.LogKeySessionID, req.GetManifestList()).Error("Please enable experimental to use manifest feature")
		return emptyResp, errors.New("please enable experimental to use manifest feature")
	}

	manifestName := req.GetManifestList()
	manifestImage := req.GetManifest()

	// get list image
	_, listImage, err := image.FindImage(b.daemon.localStore, manifestName)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Get list image err: %v", err)
		return emptyResp, err
	}

	// load list from list image
	list, err := isulamanifest.LoadListFromImage(b.daemon.localStore, listImage.ID)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Load list from image err: %v", err)
		return emptyResp, err
	}

	// add image to list, if image already exists, it will be substituted
	instanceDigest, err := list.AddImage(ctx, b.daemon.localStore, manifestImage)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Add image to list err: %v", err)
		return emptyResp, err
	}

	// update image platform if user specifies
	list.UpdateImagePlatform(req, instanceDigest)

	// save list to image
	_, err = list.SaveListToImage(b.daemon.localStore, listImage.ID, "")
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Save list to image err: %v", err)
	}

	return emptyResp, err
}

// ManifestInspect inspects manifest list
func (b *Backend) ManifestInspect(ctx context.Context, req *pb.ManifestInspectRequest) (*pb.ManifestInspectResponse, error) {
	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
	}).Info("ManifestInspectRequest received")

	if !b.daemon.opts.Experimental {
		logrus.WithField(util.LogKeySessionID, req.GetManifestList()).Error("Please enable experimental to use manifest feature")
		return &pb.ManifestInspectResponse{}, errors.New("please enable experimental to use manifest feature")
	}

	manifestName := req.GetManifestList()

	// get list image
	ref, _, err := image.FindImage(b.daemon.localStore, manifestName)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Get list image err: %v", err)
		return &pb.ManifestInspectResponse{}, err
	}

	// get image reference
	src, err := ref.NewImageSource(ctx, image.GetSystemContext())
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Get list image source err: %v", err)
		return &pb.ManifestInspectResponse{}, err
	}

	defer func() {
		if cErr := src.Close(); cErr != nil {
			logrus.Warnf("Image source closing error: %v", cErr)
		}
	}()

	// get image manifest
	manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
	if err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("Get list image manifest err: %v", err)
		return &pb.ManifestInspectResponse{}, err
	}

	// check whether image is a list image
	if !manifest.MIMETypeIsMultiImage(manifestType) {
		logrus.WithField(util.LogKeySessionID, manifestName).Errorf("%v is not a manifest list", manifestName)
		return &pb.ManifestInspectResponse{}, errors.Errorf("%v is not a manifest list", manifestName)
	}

	// return list image data
	return &pb.ManifestInspectResponse{
		Data: manifestBytes,
	}, nil
}

type manifestPushOptions struct {
	sysCtx       *types.SystemContext
	logger       *logger.Logger
	localStore   *store.Store
	manifestName string
	dest         string
}

// ManifestPush pushes manifest list to destination
func (b *Backend) ManifestPush(req *pb.ManifestPushRequest, stream pb.Control_ManifestPushServer) error {
	logrus.WithFields(logrus.Fields{
		"ManifestList": req.GetManifestList(),
		"Destination":  req.GetDest(),
	}).Info("ManifestPushRequest received")

	if !b.daemon.opts.Experimental {
		logrus.WithField(util.LogKeySessionID, req.GetManifestList()).Error("Please enable experimental to use manifest feature")
		return errors.New("please enable experimental to use manifest feature")
	}

	manifestName := req.GetManifestList()
	cliLogger := logger.NewCliLogger(constant.CliLogBufferLen)
	opt := manifestPushOptions{
		sysCtx:       image.GetSystemContext(),
		logger:       cliLogger,
		localStore:   b.daemon.localStore,
		manifestName: manifestName,
		dest:         req.GetDest(),
	}

	eg, egCtx := errgroup.WithContext(stream.Context())
	eg.Go(manifestPushHandler(egCtx, opt))
	eg.Go(manifestPushMessageHandler(stream, cliLogger))

	if err := eg.Wait(); err != nil {
		logrus.WithField(util.LogKeySessionID, manifestName).Warnf("Manifest push stream closed with: %v", err)
		return err
	}

	return nil
}

func manifestPushHandler(ctx context.Context, options manifestPushOptions) func() error {
	return func() error {
		defer options.logger.CloseContent()

		exOpts := exporter.ExportOptions{
			Ctx:                ctx,
			SystemContext:      options.sysCtx,
			ReportWriter:       options.logger,
			ManifestType:       manifest.DockerV2Schema2MediaType,
			ImageListSelection: copy.CopyAllImages,
		}

		if err := exporter.Export(options.manifestName, "manifest:"+options.dest, exOpts, options.localStore); err != nil {
			logrus.WithField(util.LogKeySessionID, options.manifestName).
				Errorf("Push manifest %s to %s failed: %v", options.manifestName, options.dest, err)
			return errors.Wrapf(err, "push manifest %s to %s failed", options.manifestName, options.dest)
		}

		return nil
	}
}

func manifestPushMessageHandler(stream pb.Control_ManifestPushServer, cliLogger *logger.Logger) func() error {
	return func() error {
		for content := range cliLogger.GetContent() {
			if content == "" {
				return nil
			}
			if err := stream.Send(&pb.ManifestPushResponse{
				Result: content,
			}); err != nil {
				return err
			}
		}

		return nil
	}
}
