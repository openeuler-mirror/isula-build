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
// Create: 2020-01-20
// Description: store related functions

// Package store provides interface for store runtime and persistent data
package store

import (
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// DaemonStoreOptions is the store options of daemon
type DaemonStoreOptions struct {
	DataRoot     string
	RunRoot      string
	Driver       string
	DriverOption []string
}

var (
	storeOpts DaemonStoreOptions
)

// Store is used to store the runtime and persistent data
type Store struct {
	// storage.Store wraps up the various types of file-based stores
	storage.Store
}

// GetDefaultStoreOptions returns default store options.
func GetDefaultStoreOptions(configOnly bool) (storage.StoreOptions, error) {
	options, err := storage.DefaultStoreOptions(false, 0)
	if err != nil {
		return storage.StoreOptions{}, err
	}

	if !configOnly {
		// StoreOpts override specific parameters of options
		if storeOpts.DataRoot != "" {
			options.GraphRoot = storeOpts.DataRoot
		}
		if storeOpts.RunRoot != "" {
			options.RunRoot = storeOpts.RunRoot
		}
		if storeOpts.Driver != "" {
			options.GraphDriverName = storeOpts.Driver
		}
		if len(storeOpts.DriverOption) > 0 {
			options.GraphDriverOptions = storeOpts.DriverOption
		}
	}

	return options, nil
}

// SetDefaultStoreOptions sets the default store options
func SetDefaultStoreOptions(opt DaemonStoreOptions) {
	if opt.DataRoot != "" {
		storeOpts.DataRoot = opt.DataRoot
	}

	if opt.RunRoot != "" {
		storeOpts.RunRoot = opt.RunRoot
	}

	if opt.Driver != "" {
		storeOpts.Driver = opt.Driver
	}

	if len(opt.DriverOption) > 0 {
		storeOpts.DriverOption = opt.DriverOption
	}
}

// SetDefaultConfigFilePath sets the default configuration to the specified path
func SetDefaultConfigFilePath(path string) {
	storage.SetDefaultConfigFilePath(path)
}

// GetStore returns a Store object.If it is called the first time,
// a store object will be created by the default store options.
func GetStore() (Store, error) {
	options, err := GetDefaultStoreOptions(false)
	if err != nil {
		return Store{}, err
	}

	store, err := storage.GetStore(options)
	if err != nil {
		return Store{}, err
	}

	is.Transport.SetStore(store)

	return Store{store}, nil
}

// CleanContainerStore unmount the containers and delete them
func (s *Store) CleanContainerStore() {
	containers, err := s.Containers()
	if err != nil {
		logrus.Warn("Failed to get containers while cleaning the container store")
		return
	}

	for _, container := range containers {
		if _, err := s.Unmount(container.ID, false); err != nil {
			logrus.Warnf("Unmount container store failed while cleaning %q", container.ID)
		}
		if err := s.DeleteContainer(container.ID); err != nil {
			logrus.Warnf("Delete container store failed while cleaning %q", container.ID)
		}
	}
}
