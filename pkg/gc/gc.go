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
 * Description: This file is used for recycling
******************************************************************************/

package gc

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RegisterOption is the register option for GC
type RegisterOption struct {
	recycleFunc func(interface{}) error
	recycleData interface{}
	name        string
	interval    time.Duration
	once        bool
}

type node struct {
	recycleFunc func(interface{}) error
	recycleData interface{}
	interval    time.Duration
	lastTrigger time.Time
	running     bool
	once        bool
	success     bool
	sync.Mutex
}

// GarbageCollector is the struct of GC, nodes record
// all recycling functions
type GarbageCollector struct {
	sync.RWMutex
	nodes map[string]*node
}

func newNode(option *RegisterOption) *node {
	return &node{
		recycleFunc: option.recycleFunc,
		recycleData: option.recycleData,
		interval:    option.interval,
		lastTrigger: time.Now(),
		once:        option.once,
	}
}

func (n *node) isDiscarded() bool {
	if n.once && n.success {
		return true
	}

	return false
}

func (n *node) isReadyToRun(now time.Time) bool {
	if n.isDiscarded() || n.running || now.Sub(n.lastTrigger) < n.interval {
		return false
	}

	return true
}

func (n *node) checkAndExec(now time.Time) {
	n.Lock()
	defer n.Unlock()

	if !n.isReadyToRun(now) {
		return
	}

	n.lastTrigger = now
	n.running = true
	err := n.recycleFunc(n.recycleData)
	if err == nil {
		n.success = true
	}
	n.running = false
}

// NewGC makes a new GC
func NewGC() *GarbageCollector {
	return &GarbageCollector{nodes: make(map[string]*node)}
}

// RegisterGC registers a recycling function
// once is false when the GC type is loop
// interval is the interval time in every loop
func (g *GarbageCollector) RegisterGC(option *RegisterOption) error {
	if option == nil {
		return errors.New("register option is nil")
	}

	g.Lock()
	defer g.Unlock()

	if _, ok := g.nodes[option.name]; ok {
		return errors.Errorf("recycle function %s has been registered", option.name)
	}

	g.nodes[option.name] = newNode(option)

	logrus.Debugf("Recycle function %s is registered successfully", option.name)

	return nil
}

// RemoveGCNode removes the GC function
func (g *GarbageCollector) RemoveGCNode(name string) {
	g.Lock()
	delete(g.nodes, name)
	g.Unlock()
	logrus.Debugf("Recycle function %s is removed successfully", name)
}

// StartGC starts the GC in a new goroutine, it will check the time to execute
// recycling function every second
func (g *GarbageCollector) StartGC(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				logrus.Debugf("GC exits now")
				return
			case now := <-tick.C:
				g.RLock()
				for name := range g.nodes {
					n := g.nodes[name]
					if n.isDiscarded() {
						go g.RemoveGCNode(name)
						continue
					}
					go n.checkAndExec(now)
				}
				g.RUnlock()
			}
		}
	}()
}
