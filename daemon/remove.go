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

	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
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
		rmFailed   bool
	)
	s := &b.daemon.localStore

	rmImageIDs = req.ImageID
	if req.All || req.Prune {
		rmImageIDs, err = getImageIDs(s, req.Prune)
		if err != nil {
			return err
		}
	}

	for _, imageID := range rmImageIDs {
		_, img, err := image.FindImageLocally(s, imageID)
		if err != nil {
			rmFailed = true
			errMsg := fmt.Sprintf("Find local image %q failed: %v", imageID, err)
			logrus.Error(errMsg)
			if err = stream.Send(&pb.RemoveResponse{LayerMessage: errMsg}); err != nil {
				return err
			}
			continue
		}

		// just untag image name if it refers to multiple tags
		if len(img.Names) > 1 {
			removed, uerr := untagImage(imageID, s, img)
			if uerr != nil {
				rmFailed = true
				errMsg := fmt.Sprintf("Untag image %q failed: %v", imageID, uerr)
				logrus.Error(errMsg)
				if err = stream.Send(&pb.RemoveResponse{LayerMessage: errMsg}); err != nil {
					return err
				}
				continue
			}

			if removed == true {
				imageString := fmt.Sprintf("Untagged image: %v", imageID)
				logrus.Debug(imageString)
				if err = stream.Send(&pb.RemoveResponse{LayerMessage: imageString}); err != nil {
					return err
				}
				continue
			}
		}

		layers, err := s.DeleteImage(img.ID, true)
		if err != nil {
			// if delete failed, print out message and continue deleting the rest images
			rmFailed = true
			errMsg := fmt.Sprintf("Remove image %q failed: %v", imageID, err)
			logrus.Error(errMsg)
			if err = stream.Send(&pb.RemoveResponse{LayerMessage: errMsg}); err != nil {
				return err
			}
			continue
		}

		for _, layer := range layers {
			layerString := fmt.Sprintf("Deleted layer: sha256:%v", layer)
			logrus.Debug(layerString)
			if err = stream.Send(&pb.RemoveResponse{LayerMessage: layerString}); err != nil {
				return err
			}
		}

		// after image is deleted successfully, print it out
		imageString := fmt.Sprintf("Deleted image: %v", imageID)
		logrus.Debug(imageString)
		if err = stream.Send(&pb.RemoveResponse{LayerMessage: imageString}); err != nil {
			return err
		}
	}

	if rmFailed {
		return errors.New("remove one or more images failed")
	}
	return nil
}

func untagImage(imageID string, store storage.Store, image *storage.Image) (bool, error) {
	newNames := make([]string, 0, 0)
	removed := false
	for _, imgName := range image.Names {
		if imgName == imageID {
			removed = true
			continue
		}
		newNames = append(newNames, imgName)
	}

	if removed == true {
		if err := store.SetNames(image.ID, newNames); err != nil {
			return false, errors.Wrapf(err, "remove name %v from image %v error", imageID, image.ID)
		}
	}

	return removed, nil
}

func getImageIDs(s *store.Store, prune bool) ([]string, error) {
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
