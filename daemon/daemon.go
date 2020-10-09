// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-01-20
// Description: This file is used for daemon setting

// Package daemon is used for isula-build daemon
package daemon

import (
	"context"
	"crypto/rsa"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/containerd/containerd/sys/reaper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder"
	"isula.org/isula-build/pkg/gc"
	"isula.org/isula-build/pkg/stack"
	"isula.org/isula-build/pkg/systemd"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

// Options carries the options configured to daemon
type Options struct {
	Debug         bool
	Group         string
	LogLevel      string
	DataRoot      string
	RunRoot       string
	StorageDriver string
	StorageOpts   []string
	RuntimePath   string
}

// Daemon struct carries the main contents in daemon
type Daemon struct {
	sync.RWMutex
	opts       *Options
	builders   map[string]builder.Builder
	entities   map[string]string
	backend    *Backend
	grpc       *GrpcServer
	localStore store.Store
	key        *rsa.PrivateKey
}

// NewDaemon new a daemon instance
func NewDaemon(opts Options, store store.Store) (*Daemon, error) {
	rsaKey, err := util.GenerateRSAKey(util.DefaultRSAKeySize)
	if err != nil {
		return nil, err
	}
	if err := util.GenRSAPublicKeyFile(rsaKey, util.DefaultRSAKeyPath); err != nil {
		return nil, err
	}

	return &Daemon{
		opts:       &opts,
		builders:   make(map[string]builder.Builder),
		entities:   make(map[string]string),
		localStore: store,
		key:        rsaKey,
	}, nil
}

// Run runs the daemon process
func (d *Daemon) Run() (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc := gc.NewGC()
	gc.StartGC(ctx)

	if rerr := d.registerSubReaper(gc); rerr != nil {
		return rerr
	}

	logrus.Debugf("Daemon start with option %#v", d.opts)

	stack.Setup(d.opts.RunRoot)

	d.NewBackend()

	if err = d.NewGrpcServer(); err != nil {
		return err
	}
	d.backend.Register(d.grpc.server)
	// after the daemon is done setting up we can notify systemd api
	systemd.NotifySystemReady()

	errCh := make(chan error)
	if err = d.grpc.Run(ctx, errCh, cancel); err != nil {
		logrus.Error("Running GRPC server failed: ", err)
	}

	select {
	case serverErr, ok := <-errCh:
		if !ok {
			logrus.Errorf("Channel errCh closed, check grpc server err")
		}
		err = serverErr
		cancel()
	// channel closed is what we expected since it's daemon normal behavior
	case <-ctx.Done():
		logrus.Infof("Context finished with: %v", ctx.Err())
	}

	systemd.NotifySystemStopping()
	d.grpc.server.GracefulStop()
	d.backend.wg.Wait()
	return err
}

// NewBuilder returns the builder with request sent from GRPC service
func (d *Daemon) NewBuilder(ctx context.Context, req *pb.BuildRequest) (b builder.Builder, err error) {
	// buildDir is used to set directory which is used to store data
	buildDir := filepath.Join(d.opts.DataRoot, req.BuildID)
	// runDir is used to store such as container bundle directories
	runDir := filepath.Join(d.opts.RunRoot, req.BuildID)

	// this key with BuildDir will be used by exporter to save blob temporary
	// NOTE: keep it be updated before NewBuilder. ctx will be taken by Builder
	ctx = context.WithValue(ctx, util.BuildDirKey(util.BuildDir), buildDir)
	b, err = builder.NewBuilder(ctx, d.localStore, req, d.opts.RuntimePath, buildDir, runDir, d.key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to new builder")
	}

	d.Lock()
	defer d.Unlock()
	entityID := b.EntityID()
	if buildID, exist := d.entities[entityID]; exist {
		return nil, errors.Errorf("the dockerfile is already on building with static build mode by buildID: %s", buildID)
	}
	d.entities[entityID] = req.BuildID
	d.builders[req.BuildID] = b

	return b, nil
}

// Builder returns an Builder to caller. Caller should check the return value if it is nil
func (d *Daemon) Builder(buildID string) (builder.Builder, error) {
	d.RLock()
	defer d.RUnlock()
	if _, ok := d.builders[buildID]; !ok {
		return nil, errors.Errorf("could not find builder with build job %s", buildID)
	}
	return d.builders[buildID], nil
}

// deleteBuilder deletes builder from daemon
func (d *Daemon) deleteBuilder(buildID string) {
	d.Lock()
	builder := d.builders[buildID]
	delete(d.builders, buildID)
	delete(d.entities, builder.EntityID())
	d.Unlock()
}

// deleteAllBuilders deletes all Builders stored in daemon
func (d *Daemon) deleteAllBuilders() {
	d.Lock()
	d.builders = make(map[string]builder.Builder)
	d.entities = make(map[string]string)
	d.Unlock()
}

// Cleanup cleans the resource
func (d *Daemon) Cleanup() error {
	if d.backend != nil {
		d.backend.deleteAllStatus()
	}
	if err := os.Remove(util.DefaultRSAKeyPath); err != nil {
		logrus.Info("Delete key failed")
	}
	d.deleteAllBuilders()
	d.localStore.CleanContainers()
	_, err := d.localStore.Shutdown(false)
	return err
}

func (d *Daemon) registerSubReaper(g *gc.GarbageCollector) error {
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0); err != nil { //nolint, gomod
		return errors.Errorf("set subreaper failed: %v", err)
	}

	childProcessReap := func(i interface{}) error {
		var err error

		daemonTmp := i.(*Daemon)
		daemonTmp.Lock()
		defer daemonTmp.Unlock()

		// if any of image build process is running, skip reap
		if len(daemonTmp.builders) != 0 {
			return nil
		}
		if err = reaper.Reap(); err != nil {
			logrus.Errorf("Reap child process error: %v", err)
		}
		return err
	}

	opt := &gc.RegisterOption{
		Name:        "subReaper",
		Interval:    10 * time.Second,
		RecycleData: d,
		RecycleFunc: childProcessReap,
	}

	return g.RegisterGC(opt)
}
