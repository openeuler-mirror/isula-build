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
// Create: 2020-06-20
// Description: This file is used for recycling test

package gc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

type mockDaemon struct {
	backend *mockBackend
	opts    string
	sync.RWMutex
}

type mockBackend struct {
	status map[string]string
}

var (
	backend, emptyBackend *mockBackend
	gc                    *GarbageCollector
)

func init() {
	backend = &mockBackend{status: map[string]string{"init": "init"}}
	emptyBackend = &mockBackend{}
	ctx, _ := context.WithCancel(context.Background())
	gc = NewGC()
	gc.StartGC(ctx)
}

func TestRegisterGCWithEmptyOption(t *testing.T) {
	err := gc.RegisterGC(nil)
	assert.ErrorContains(t, err, "is nil")
}

func TestGCRoutineExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	gcExit := NewGC()
	gcExit.StartGC(ctx)
	d := &mockDaemon{backend: emptyBackend}
	f := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.backend = backend
		return nil
	}
	registerOption := &RegisterOption{
		Name:        "routineExit",
		RecycleFunc: f,
		RecycleData: d,
		Interval:    2 * time.Second,
	}
	err := gcExit.RegisterGC(registerOption)
	assert.NilError(t, err)

	// cancel it and the resource will not be recycled
	cancel()
	time.Sleep(3 * time.Second)
	assert.Equal(t, d.backend, emptyBackend)
	gcExit.RemoveGCNode(registerOption.Name)
}

func TestLoopGC(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.Lock()
		daemon.backend = emptyBackend
		daemon.Unlock()
		return nil
	}
	registerOption := &RegisterOption{
		Name:        "recycleBackendResource",
		RecycleFunc: f,
		RecycleData: d,
		Interval:    time.Second,
	}

	// register recycleBackendStore and the backend resource will be released
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)
	d.RLock()
	b := d.backend
	d.RUnlock()
	assert.Equal(t, b, emptyBackend)

	// the same Name has been registered
	err = gc.RegisterGC(registerOption)
	assert.ErrorContains(t, err, "has been registered")

	// remove success and the resource won't be recycled again
	gc.RemoveGCNode(registerOption.Name)
	d.backend = backend
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, backend)

	// new recycle func will be registered success
	f2 := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.Lock()
		daemon.opts = "ok"
		daemon.Unlock()
		return nil
	}
	registerOption.Name = "recycleOpts"
	registerOption.RecycleFunc = f2
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)
	d.RLock()
	opts := d.opts
	d.RUnlock()
	assert.Equal(t, opts, "ok")
	gc.RemoveGCNode(registerOption.Name)
}

func TestOnceGC(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		return errors.New("recycle failed")
	}
	registerOption := &RegisterOption{
		Name:        "recycleTestErr",
		RecycleFunc: f,
		RecycleData: d,
		Interval:    time.Second,
		Once:        true,
	}
	// register a gc which will always return error
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)

	// register will failed
	err = gc.RegisterGC(registerOption)
	assert.ErrorContains(t, err, "has been registered")
	gc.RemoveGCNode(registerOption.Name)

	d.backend = &mockBackend{status: map[string]string{"Once": "Once"}}
	// register a normal gc
	f2 := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.Lock()
		daemon.backend = emptyBackend
		daemon.Unlock()
		return nil
	}
	registerOption.Name = "Once"
	registerOption.RecycleFunc = f2
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(4 * time.Second)
	d.RLock()
	b := d.backend
	d.RUnlock()
	assert.Equal(t, b, emptyBackend)

	// the last normal gc has been removed after executing,
	// new gc will be registered successfully
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	gc.RemoveGCNode(registerOption.Name)
}

func TestGCAlreadyInRunning(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.Lock()
		daemon.backend = emptyBackend
		daemon.Unlock()
		time.Sleep(30 * time.Second)
		return nil
	}
	registerOption := &RegisterOption{
		Name:        "recycleSlow",
		RecycleFunc: f,
		RecycleData: d,
		Interval:    time.Second,
	}
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)

	time.Sleep(2 * time.Second)
	d.RLock()
	b := d.backend
	d.RUnlock()
	assert.Equal(t, b, emptyBackend)

	// a recycling work is doing and it won't be triggered now
	d.RLock()
	d.backend = backend
	d.RUnlock()
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, backend)
	gc.RemoveGCNode(registerOption.Name)
}
