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
// Create: 2020-08-03
// Description: This file is "info" command for backend

package daemon

import (
	"context"
	"runtime"

	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/containers/storage/pkg/system"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/image"
)

// Info to get isula-build system information
func (b *Backend) Info(ctx context.Context, req *pb.InfoRequest) (*pb.InfoResponse, error) {
	logrus.Info("InfoRequest received")

	// get memory information
	sysMemInfo, err := system.ReadMemInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "get memory information err")
	}

	// get storage backing file system information
	var storageBackingFs string
	status, err := b.daemon.localStore.Status()
	if err != nil {
		return nil, errors.Wrapf(err, "get storage status info err")
	}
	for _, pair := range status {
		if pair[0] == "Backing Filesystem" {
			storageBackingFs = pair[1]
		}
	}

	// get registry information
	registriesSearch, registriesInsecure, registriesBlock, err := getRegistryInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "get registries info err")
	}

	memInfo := &pb.MemData{
		MemTotal:  sysMemInfo.MemTotal,
		MemFree:   sysMemInfo.MemFree,
		SwapTotal: sysMemInfo.SwapTotal,
		SwapFree:  sysMemInfo.SwapFree,
	}
	storageInfo := &pb.StorageData{
		StorageDriver:    b.daemon.localStore.GraphDriverName(),
		StorageBackingFs: storageBackingFs,
	}
	registryInfo := &pb.RegistryData{
		RegistriesSearch:   registriesSearch,
		RegistriesInsecure: registriesInsecure,
		RegistriesBlock:    registriesBlock,
	}

	// generate info response
	infoResponse := &pb.InfoResponse{
		MemInfo:      memInfo,
		MemStat:      nil,
		StorageInfo:  storageInfo,
		RegistryInfo: registryInfo,
		DataRoot:     b.daemon.opts.DataRoot,
		RunRoot:      b.daemon.opts.RunRoot,
		OCIRuntime:   b.daemon.opts.RuntimePath,
		BuilderNum:   int64(len(b.daemon.builders)),
		GoRoutines:   int64(runtime.NumGoroutine()),
		Experimental: b.daemon.opts.Experimental,
	}

	if req.Verbose {
		var rms runtime.MemStats
		// get runtime mem stats
		runtime.ReadMemStats(&rms)
		memStat := &pb.MemStat{
			MemSys:       rms.Sys,
			HeapSys:      rms.HeapSys,
			HeapAlloc:    rms.HeapAlloc,
			HeapInUse:    rms.HeapInuse,
			HeapIdle:     rms.HeapIdle,
			HeapReleased: rms.HeapReleased,
		}
		infoResponse.MemStat = memStat
	}

	// default OCI runtime is "runc"
	if b.daemon.opts.RuntimePath == "" {
		infoResponse.OCIRuntime = "runc"
	}

	return infoResponse, nil
}

func getRegistryInfo() ([]string, []string, []string, error) {
	registriesInsecure := make([]string, 0, 0)
	registriesBlock := make([]string, 0, 0)
	systemContext := image.GetSystemContext()

	registriesSearch, err := sysregistriesv2.UnqualifiedSearchRegistries(systemContext)
	if err != nil {
		return nil, nil, nil, err
	}

	registries, err := sysregistriesv2.GetRegistries(systemContext)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, registry := range registries {
		if registry.Insecure {
			registriesInsecure = append(registriesInsecure, registry.Prefix)
		}
		if registry.Blocked {
			registriesBlock = append(registriesBlock, registry.Prefix)
		}
	}

	return registriesSearch, registriesInsecure, registriesBlock, nil
}
