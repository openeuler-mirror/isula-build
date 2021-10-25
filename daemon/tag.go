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
// Create: 2020-07-20
// Description: This file is "tag" command for backend

package daemon

import (
	"context"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
)

// Tag adds an additional tag to an image
func (b *Backend) Tag(ctx context.Context, req *pb.TagRequest) (*gogotypes.Empty, error) {
	logrus.WithFields(logrus.Fields{
		"Image": req.GetImage(),
		"Tag":   req.GetTag(),
	}).Info("TagRequest received")

	var emptyResp = &gogotypes.Empty{}
	s := b.daemon.localStore

	_, img, err := image.FindImage(s, req.Image)
	if err != nil {
		return emptyResp, errors.Wrapf(err, "find local image %v error", req.Image)
	}

	_, imageName, err := image.GetNamedTaggedReference(req.Tag)
	if err != nil {
		return emptyResp, err
	}

	if err := s.SetNames(img.ID, append(img.Names, imageName)); err != nil {
		return emptyResp, errors.Wrapf(err, "set name %v to image %q error", req.Tag, req.Image)
	}

	return emptyResp, nil
}
