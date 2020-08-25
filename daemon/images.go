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
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

const (
	none = "<none>"
)

// List lists all images
func (b *Backend) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	logrus.WithFields(logrus.Fields{
		"ImageName": req.GetImageName(),
	}).Info("ListRequest received")

	var reqRepository, reqTag string
	const minImageFieldLenWithTag = 2
	if req.ImageName != "" {
		imageName := req.ImageName
		_, img, err := image.FindImage(b.daemon.localStore, imageName)
		if err != nil {
			return nil, errors.Wrapf(err, "find local image %v error", imageName)
		}

		parts := strings.Split(imageName, ":")
		if len(parts) >= minImageFieldLenWithTag {
			reqRepository, reqTag = strings.Join(parts[0:len(parts)-1], ":"), parts[len(parts)-1]
		}

		imageInfo := &pb.ListResponse_ImageInfo{
			Repository: reqRepository,
			Tag:        reqTag,
			Id:         img.ID,
			Created:    img.Created.Format(constant.LayoutTime),
			Size_:      getImageSize(&b.daemon.localStore, img.ID),
		}

		return &pb.ListResponse{Images: []*pb.ListResponse_ImageInfo{imageInfo}}, nil
	}

	images, err := b.daemon.localStore.Images()
	if err != nil {
		return &pb.ListResponse{}, errors.Wrap(err, "failed list images from local storage")
	}
	sort.Slice(images, func(i, j int) bool {
		return images[i].Created.After(images[j].Created)
	})
	result := make([]*pb.ListResponse_ImageInfo, 0, len(images))
	for _, image := range images {
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
			if reqRepository != "" && reqRepository != repository {
				continue
			}
			if reqTag != "" && reqTag != tag {
				continue
			}

			imageInfo := &pb.ListResponse_ImageInfo{
				Repository: repository,
				Tag:        tag,
				Id:         image.ID,
				Created:    image.Created.Format(constant.LayoutTime),
				Size_:      getImageSize(&b.daemon.localStore, image.ID),
			}
			result = append(result, imageInfo)
		}
	}
	return &pb.ListResponse{Images: result}, nil
}

func getImageSize(store *store.Store, id string) string {
	imgSize, err := store.ImageSize(id)
	if err != nil {
		imgSize = -1
	}
	return util.FormatSize(float64(imgSize))
}
