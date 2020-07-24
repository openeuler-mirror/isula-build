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
 * Create: 2020-06-20
 * Description: This file is used for recycling test
******************************************************************************/

package gc

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/assert"
)

type mockDaemon struct {
	backend *mockBackend
	opts    string
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
		name:        "routineExit",
		recycleFunc: f,
		recycleData: d,
		interval:    2 * time.Second,
	}
	err := gcExit.RegisterGC(registerOption)
	assert.NilError(t, err)

	// cancel it and the resource will not be recycled
	cancel()
	time.Sleep(3 * time.Second)
	assert.Equal(t, d.backend, emptyBackend)
	gcExit.RemoveGCNode(registerOption.name)
}

func TestLoopGC(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.backend = emptyBackend
		return nil
	}
	registerOption := &RegisterOption{
		name:        "recycleBackendResource",
		recycleFunc: f,
		recycleData: d,
		interval:    time.Second,
	}

	// register recycleBackendStore and the backend resource will be released
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, emptyBackend)

	// the same name has been registered
	err = gc.RegisterGC(registerOption)
	assert.ErrorContains(t, err, "has been registered")

	// remove success and the resource won't be recycled again
	gc.RemoveGCNode(registerOption.name)
	d.backend = backend
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, backend)

	// new recycle func will be registered success
	f2 := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.opts = "ok"
		return nil
	}
	registerOption.name = "recycleOpts"
	registerOption.recycleFunc = f2
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.opts, "ok")
	gc.RemoveGCNode(registerOption.name)
}

func TestOnceGC(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		return errors.New("recycle failed")
	}
	registerOption := &RegisterOption{
		name:        "recycleTestErr",
		recycleFunc: f,
		recycleData: d,
		interval:    time.Second,
		once:        true,
	}
	// register a gc which will always return error
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(2 * time.Second)

	// register will failed
	err = gc.RegisterGC(registerOption)
	assert.ErrorContains(t, err, "has been registered")
	gc.RemoveGCNode(registerOption.name)

	d.backend = &mockBackend{status: map[string]string{"once": "once"}}
	// register a normal gc
	f2 := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.backend = emptyBackend
		return nil
	}
	registerOption.name = "once"
	registerOption.recycleFunc = f2
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	time.Sleep(4 * time.Second)
	assert.Equal(t, d.backend, emptyBackend)

	// the last normal gc has been removed after executing,
	// new gc will be registered successfully
	err = gc.RegisterGC(registerOption)
	assert.NilError(t, err)
	gc.RemoveGCNode(registerOption.name)
}

func TestGCAlreadyInRunning(t *testing.T) {
	d := &mockDaemon{backend: backend}
	f := func(i interface{}) error {
		daemon := i.(*mockDaemon)
		daemon.backend = emptyBackend
		time.Sleep(30 * time.Second)
		return nil
	}
	registerOption := &RegisterOption{
		name:        "recycleSlow",
		recycleFunc: f,
		recycleData: d,
		interval:    time.Second,
	}
	err := gc.RegisterGC(registerOption)
	assert.NilError(t, err)

	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, emptyBackend)

	// a recycling work is doing and it won't be triggered now
	d.backend = backend
	time.Sleep(2 * time.Second)
	assert.Equal(t, d.backend, backend)
	gc.RemoveGCNode(registerOption.name)
}
