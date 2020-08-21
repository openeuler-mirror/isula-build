// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jingxiao Lu
// Create: 2020-03-20
// Description: Builder related functions

// Package builder includes Builder related functions
package builder

import (
	"context"

	"github.com/pkg/errors"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/exporter"
	"isula.org/isula-build/store"
)

// Builder is an interface for building an image
type Builder interface {
	Build() (imageID string, err error)
	StatusChan() <-chan string
	CleanResources() error
	OutputPipeWrapper() *exporter.PipeWrapper
	EntityID() string
}

// NewBuilder init a builder
func NewBuilder(ctx context.Context, store store.Store, req *pb.BuildRequest, runtimePath, buildDir, runDir string) (Builder, error) {
	switch req.GetBuildType() {
	case constant.BuildContainerImageType:
		return dockerfile.NewBuilder(ctx, store, req, runtimePath, buildDir, runDir)
	default:
		return nil, errors.Errorf("the build type %q is not supported", req.GetBuildType())
	}
}
