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
	"sync"

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
	sync.RWMutex
}

// SetStorageConfigFilePath sets the default file path of storage configuration
func SetStorageConfigFilePath(path string) {
	storage.SetDefaultConfigFilePath(path)
}

// GetStorageConfigFileOptions returns the default storage config options.
func GetStorageConfigFileOptions() (storage.StoreOptions, error) {
	options, err := storage.DefaultStoreOptions(false, 0)
	if err != nil {
		return storage.StoreOptions{}, err
	}

	return options, nil
}

// SetDefaultStoreOptions sets the default store options
func SetDefaultStoreOptions(opt DaemonStoreOptions) {
	storeOpts = opt
}

// GetDefaultStoreOptions returns default store options.
func GetDefaultStoreOptions() (storage.StoreOptions, error) {
	options, err := storage.DefaultStoreOptions(false, 0)
	if err != nil {
		return storage.StoreOptions{}, err
	}

	options.GraphRoot = storeOpts.DataRoot
	options.RunRoot = storeOpts.RunRoot
	options.GraphDriverName = storeOpts.Driver
	options.GraphDriverOptions = storeOpts.DriverOption

	return options, nil
}

// GetStore returns a Store object. If it is called the first time,
// a store object will be created by the default store options.
func GetStore() (Store, error) {
	options, err := GetDefaultStoreOptions()
	if err != nil {
		return Store{}, err
	}

	store, err := storage.GetStore(options)
	if err != nil {
		return Store{}, err
	}

	is.Transport.SetStore(store)

	return Store{Store: store}, nil
}

// CleanContainers unmount the containers and delete them
func (s *Store) CleanContainers() {
	containers, err := s.Containers()
	if err != nil {
		logrus.Warn("Failed to get containers while cleaning the container store")
		return
	}

	for _, container := range containers {
		if cerr := s.CleanContainer(container.ID); cerr != nil {
			logrus.Warnf("Clean container %q failed", container.ID)
		}
	}
}

// CleanContainer cleans the container in store
func (s *Store) CleanContainer(id string) error {
	s.Lock()
	defer s.Unlock()

	// Do not care about all the errors whiling cleaning the container,
	// just return one if the error occurs.
	var finalErr error
	if _, uerr := s.Unmount(id, false); uerr != nil {
		finalErr = uerr
		logrus.Warnf("Unmount container store failed while cleaning %q", id)
	}
	if derr := s.DeleteContainer(id); derr != nil {
		finalErr = derr
		logrus.Warnf("Delete container store failed while cleaning %q", id)
	}

	return finalErr
}
