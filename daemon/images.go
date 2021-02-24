// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu
// Create: 2020-01-20
// Description: This file is "images" command for backend

package daemon

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const (
	none                    = "<none>"
	decimalPrefixBase       = 1000
	minImageFieldLenWithTag = 2
)

type listOptions struct {
	localStore *store.Store
	logEntry   *logrus.Entry
	imageName  string
}

func (b *Backend) getListOptions(req *pb.ListRequest) listOptions {
	return listOptions{
		localStore: b.daemon.localStore,
		logEntry:   logrus.WithFields(logrus.Fields{"ImageName": req.GetImageName()}),
		imageName:  req.GetImageName(),
	}
}

// List lists all images
func (b *Backend) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	logrus.WithFields(logrus.Fields{
		"ImageName": req.GetImageName(),
	}).Info("ListRequest received")

	opts := b.getListOptions(req)

	slashLastIndex := strings.LastIndex(opts.imageName, "/")
	colonLastIndex := strings.LastIndex(opts.imageName, ":")
	if opts.imageName != "" && strings.Contains(opts.imageName, ":") && colonLastIndex > slashLastIndex {
		return listOneImage(opts)
	}
	return listImages(opts)
}

func listOneImage(opts listOptions) (*pb.ListResponse, error) {
	_, image, err := image.FindImage(opts.localStore, opts.imageName)
	if err != nil {
		opts.logEntry.Error(err)
		return nil, errors.Wrapf(err, "find local image %v error", opts.imageName)
	}

	result := make([]*pb.ListResponse_ImageInfo, 0, len(image.Names))
	appendImageToResult(&result, image, opts.localStore)

	for _, info := range result {
		if opts.imageName == fmt.Sprintf("%s:%s", info.Repository, info.Tag) {
			result = []*pb.ListResponse_ImageInfo{info}
		}
	}

	return &pb.ListResponse{Images: result}, nil
}

func listImages(opts listOptions) (*pb.ListResponse, error) {
	images, err := opts.localStore.Images()
	if err != nil {
		opts.logEntry.Error(err)
		return &pb.ListResponse{}, errors.Wrap(err, "failed list images from local storage")
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Created.After(images[j].Created)
	})
	result := make([]*pb.ListResponse_ImageInfo, 0, len(images))
	for i := range images {
		appendImageToResult(&result, &images[i], opts.localStore)
	}

	if opts.imageName == "" {
		return &pb.ListResponse{Images: result}, nil
	}

	sameRepositoryResult := make([]*pb.ListResponse_ImageInfo, 0, len(images))
	for _, info := range result {
		if opts.imageName == info.Repository || strings.HasPrefix(info.Id, opts.imageName) {
			sameRepositoryResult = append(sameRepositoryResult, info)
		}
	}

	if len(sameRepositoryResult) == 0 {
		return &pb.ListResponse{}, errors.Errorf("failed to list images with repository %q in local storage", opts.imageName)
	}
	return &pb.ListResponse{Images: sameRepositoryResult}, nil
}

func appendImageToResult(result *[]*pb.ListResponse_ImageInfo, image *storage.Image, store *store.Store) {
	names := image.Names
	if len(names) == 0 {
		names = []string{none}
	}

	for _, name := range names {
		repository, tag := name, none
		parts := strings.Split(name, ":")
		if len(parts) >= minImageFieldLenWithTag {
			repository, tag = strings.Join(parts[0:len(parts)-1], ":"), parts[len(parts)-1]
		}

		imageInfo := &pb.ListResponse_ImageInfo{
			Repository: repository,
			Tag:        tag,
			Id:         image.ID,
			Created:    image.Created.Format(constant.LayoutTime),
			Size_:      getImageSize(store, image.ID),
		}
		*result = append(*result, imageInfo)
	}
}

func getImageSize(store *store.Store, id string) string {
	imgSize, err := store.ImageSize(id)
	if err != nil {
		imgSize = -1
	}
	return util.FormatSize(float64(imgSize), decimalPrefixBase)
}
