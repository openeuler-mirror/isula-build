/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
 * isula-build licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: Feiyu Yang
 * Create: 2020-07-17
 * Description: This file is used for image load command
******************************************************************************/

package daemon

import (
	"context"

	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
	"isula.org/isula-build/pkg/logger"
	"isula.org/isula-build/util"
)

// Load loads the image
func (b *Backend) Load(ctx context.Context, req *pb.LoadRequest) (*pb.LoadResponse, error) {
	logrus.Info("LoadRequest received")
	if err := util.CheckLoadFile(req.Path); err != nil {
		return &pb.LoadResponse{}, err
	}

	_, si, err := image.ResolveFromImage(&image.PrepareImageOptions{
		Ctx:           ctx,
		FromImage:     "docker-archive:" + req.Path,
		SystemContext: image.GetSystemContext(),
		Store:         b.daemon.localStore,
		Reporter:      logger.NewCliLogger(constant.CliLogBufferLen),
	})
	if err != nil {
		return nil, err
	}

	if err := b.daemon.localStore.SetNames(si.ID, []string{"<none>:<none>"}); err != nil {
		return nil, err
	}

	logrus.Infof("Loaded image as %v", si.ID)
	return &pb.LoadResponse{ImageID: si.ID}, nil
}
