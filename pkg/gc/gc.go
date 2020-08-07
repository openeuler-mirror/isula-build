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
// Description: This file is used for recycling

// Package gc provides garbage collectors
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
	// RecycleFunc is the function that can recycle the resource
	RecycleFunc func(interface{}) error
	// RecycleData indicates the data to be recycled
	RecycleData interface{}
	// Name indicates the node name
	Name string
	// Interval indicates the recycle interval
	Interval time.Duration
	// Once is true when it is a once time recycle
	Once bool
}

type node struct {
	garbageCollector *GarbageCollector
	name             string
	recycleFunc      func(interface{}) error
	recycleData      interface{}
	interval         time.Duration
	lastTrigger      time.Time
	running          bool
	once             bool
	success          bool
	sync.Mutex
}

// GarbageCollector is the struct of GC, nodes record
// all recycling functions
type GarbageCollector struct {
	sync.RWMutex
	nodes map[string]*node
}

func newNode(option *RegisterOption, g *GarbageCollector) *node {
	return &node{
		garbageCollector: g,
		name:             option.Name,
		recycleFunc:      option.RecycleFunc,
		recycleData:      option.RecycleData,
		interval:         option.Interval,
		lastTrigger:      time.Now(),
		once:             option.Once,
	}
}

func (n *node) isDiscarded() bool {
	if n.once && n.success {
		return true
	}

	return false
}

func (n *node) isReadyToRun(now time.Time) bool {
	if n.running || now.Sub(n.lastTrigger) < n.interval {
		return false
	}

	return true
}

func (n *node) checkAndExec(now time.Time) {
	n.Lock()
	defer n.Unlock()

	if n.isDiscarded() {
		go n.garbageCollector.RemoveGCNode(n.name)
		return
	}

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
// Once is false when the GC type is loop
// Interval is the Interval time in every loop
func (g *GarbageCollector) RegisterGC(option *RegisterOption) error {
	if option == nil {
		return errors.New("register option is nil")
	}

	g.Lock()
	defer g.Unlock()

	if _, ok := g.nodes[option.Name]; ok {
		return errors.Errorf("recycle function %s has been registered", option.Name)
	}

	g.nodes[option.Name] = newNode(option, g)

	logrus.Infof("Recycle function %s is registered successfully", option.Name)

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
			case _, ok := <-ctx.Done():
				if !ok {
					logrus.Warnf("Context channel has been closed")
				}
				logrus.Debugf("GC exits now")
				return
			case now, ok := <-tick.C:
				if !ok {
					logrus.Warnf("Time tick channel has been closed")
					return
				}
				g.RLock()
				for name := range g.nodes {
					n := g.nodes[name]
					go n.checkAndExec(now)
				}
				g.RUnlock()
			}
		}
	}()
}
