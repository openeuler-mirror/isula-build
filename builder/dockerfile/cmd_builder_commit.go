// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Feiyu Yang
// Create: 2020-03-20
// Description: commit related functions

package dockerfile

import (
	"context"
	"encoding/json"
	"strings"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"

	transc "isula.org/isula-build/builder/dockerfile/container"
	"isula.org/isula-build/image"
	"isula.org/isula-build/util"
)

// GetPolicyContext returns a specied policy context
func GetPolicyContext() (*signature.PolicyContext, error) {
	systemContext := image.GetSystemContext()
	systemContext.DirForceCompress = true
	commitPolicy, err := signature.DefaultPolicy(systemContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the default policy by new system context")
	}

	commitPolicy.Transports[is.Transport.Name()] = signature.PolicyTransportScopes{
		"": []signature.PolicyRequirement{
			signature.NewPRInsecureAcceptAnything(),
		},
	}

	return signature.NewPolicyContext(commitPolicy)
}

func (c *cmdBuilder) newContainerReference(exporting bool) (transc.Reference, error) {
	var name reference.Named
	container, err := c.stage.localStore.Container(c.stage.containerID)
	if err != nil {
		return transc.Reference{}, errors.Wrapf(err, "error locating container %q", c.stage.containerID)
	}
	if len(container.Names) > 0 {
		if parsed, err2 := reference.ParseNamed(container.Names[0]); err2 == nil {
			name = parsed
		}
	}

	dconfig, err := json.Marshal(&c.stage.docker)
	if err != nil {
		return transc.Reference{}, errors.Wrapf(err, "error encoding docker-format image configuration %#v", c.stage.docker)
	}

	createdBy := strings.Join(util.CopyStrings(c.stage.docker.Config.Shell), " ")
	if createdBy == "" {
		createdBy = defaultShell
	}

	metadata := &transc.ReferenceMetadata{
		Name:      name,
		CreatedBy: createdBy,
		Dconfig:   dconfig,
		// container id used in the image has no meaning here,
		// so we use dockerfileDigest to fill it for distinguishing whether an image is
		// built from the same dockerfile
		ContainerID: c.stage.builder.dockerfileDigest,
		BuildTime:   c.stage.builder.buildTime,
		LayerID:     container.LayerID,
	}
	result := transc.NewContainerReference(c.stage.localStore, metadata, exporting)

	return result, nil
}

func (c *cmdBuilder) isFromImageExist(storeT is.StoreTransport) bool {
	fromImageID := c.stage.fromImageID
	if fromImageID == "" {
		return false
	}
	ref, err := storeT.ParseReference(fromImageID)
	if ref == nil || err != nil {
		return false
	}
	if img, err := storeT.GetImage(ref); img != nil && err == nil {
		return true
	}
	return false
}

func (c *cmdBuilder) commit(ctx context.Context) (string, error) {
	commitTimer := c.stage.builder.cliLog.StartTimer("COMMIT")
	tmpName := stringid.GenerateRandomID() + "-commit-tmp"
	dest, err := is.Transport.ParseStoreReference(c.stage.localStore, tmpName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create ref using %q", tmpName)
	}

	// if fromImage exist in store, set exporting false to avoid store fromImage again
	exporting := true
	if storeTransport, ok := dest.Transport().(is.StoreTransport); ok {
		exporting = !c.isFromImageExist(storeTransport)
	}

	policyContext, err := GetPolicyContext()
	if err != nil {
		return "", err
	}
	c.stage.builder.Logger().Debugf("CmdBuilder commit %q gets CommitPolicyContext OK", tmpName)
	defer func() {
		if derr := policyContext.Destroy(); derr != nil {
			c.stage.builder.Logger().Warningf("Destroy commit policy context failed: %v", derr)
		}
	}()

	// New a container image ref for copying
	srcContainerReference, err := c.newContainerReference(exporting)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create container image ref for container %q", c.stage.containerID)
	}

	imageCopyOptions := image.NewImageCopyOptions(c.stage.builder.cliLog)

	if _, err = cp.Image(ctx, policyContext, dest, &srcContainerReference, imageCopyOptions); err != nil {
		return "", errors.Wrapf(err, "error copying layers and metadata for container %q", c.stage.containerID)
	}

	img, err := is.Transport.GetStoreImage(c.stage.localStore, dest)
	if err != nil {
		return "", errors.Wrapf(err, "error locating image %q in local storage", transports.ImageName(dest))
	}

	// Remove tmp name
	newNames := util.CopyStringsWithoutSpecificElem(img.Names, tmpName)
	if err = c.stage.localStore.SetNames(img.ID, newNames); err != nil {
		return img.ID, errors.Wrapf(err, "failed to prune temporary name from image %q", img.ID)
	}
	c.stage.builder.Logger().Debugf("Reassigned names %v to image %q", newNames, img.ID)

	// Update the dest ref
	_, err = is.Transport.ParseStoreReference(c.stage.localStore, "@"+img.ID)
	if err != nil {
		return img.ID, errors.Wrapf(err, "failed to create ref using %q", img.ID)
	}
	c.stage.builder.cliLog.StopTimer(commitTimer)
	c.stage.builder.Logger().Debugln(c.stage.builder.cliLog.GetCmdTime(commitTimer))
	c.stage.builder.cliLog.Print("Committed stage %s with ID: %s\n", c.stage.name, img.ID)
	return img.ID, nil
}
