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
// Create: 2020-01-20
// Description: This file is "remove" command for backend

package daemon

import (
	"fmt"

	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/store"
)

// Remove to remove store images
func (b *Backend) Remove(req *pb.RemoveRequest, stream pb.Control_RemoveServer) error {
	logrus.WithFields(logrus.Fields{
		"ImageID": req.GetImageID(),
		"All":     req.GetAll(),
		"Prune":   req.GetPrune(),
	}).Info("RemoveRequest received")

	var (
		rmImageIDs []string
		err        error
	)
	s := b.daemon.localStore

	rmImageIDs = req.ImageID
	if req.All || req.Prune {
		rmImageIDs, err = getImageIDs(s, req.Prune)
		if err != nil {
			return err
		}
	}

	for _, imageID := range rmImageIDs {
		layers, err := s.DeleteImage(imageID, true)
		if err != nil {
			// if delete failed, print out message and continue deleting the rest images
			errMsg := fmt.Sprintf("Remove image %s failed: %v", imageID, err.Error())
			logrus.Error(errMsg)
			if err = stream.Send(&pb.RemoveResponse{LayerMessage: errMsg}); err != nil {
				return err
			}
			continue
		}

		for _, layer := range layers {
			layerString := fmt.Sprintf("Deleted: sha256:%v", layer)
			logrus.Debug(layerString)
			if err = stream.Send(&pb.RemoveResponse{LayerMessage: layerString}); err != nil {
				return err
			}
		}
	}

	return nil
}

func getImageIDs(s store.Store, prune bool) ([]string, error) {
	images, err := s.Images()
	if err != nil {
		return nil, err
	}

	var imageIDs []string
	for _, image := range images {
		if prune && len(image.Names) != 0 {
			continue
		}
		imageIDs = append(imageIDs, image.ID)
	}

	return imageIDs, nil
}
