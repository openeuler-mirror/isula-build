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
// Description: This file is used for grpc daemon setting

package daemon

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/containerd/containerd/sys"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	constant "isula.org/isula-build"
	"isula.org/isula-build/util"
)

// GrpcServer struct carries the main contents in GRPC server
type GrpcServer struct {
	server   *grpc.Server
	listener net.Listener
	path     string
}

// NewGrpcServer creates a new GRPC socket with default GRPC socket address
func (d *Daemon) NewGrpcServer() error {
	socket, path, err := newSocket(d.opts.Group)
	if err != nil {
		return errors.Errorf("create new GRPC socket failed: %v", err)
	}

	server := grpc.NewServer()
	d.grpc = &GrpcServer{
		listener: socket,
		path:     path,
		server:   server,
	}
	return nil
}

// Run running the GRPC server and collects signals for handling
func (g *GrpcServer) Run(ctx context.Context, ch chan error, cancelFunc context.CancelFunc) error {
	eg, gctx := errgroup.WithContext(ctx)
	// signal trap
	eg.Go(func() error {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT)
		select {
		case sig, ok := <-signalChannel:
			if !ok {
				logrus.Errorf("Signal chan closed")
				return nil
			}
			if sig == syscall.SIGTERM || sig == syscall.SIGINT {
				logrus.Infof("Signal %v received and exit", sig)
				g.server.Stop()
				cancelFunc()
			}
		case <-gctx.Done():
			return gctx.Err()
		}
		return nil
	})

	// grpc serve
	eg.Go(func() error {
		logrus.Infof("isula-builder is listening on %s", g.path)
		return g.server.Serve(g.listener)
	})

	go func() {
		ch <- eg.Wait()
	}()

	return nil
}

func newSocket(group string) (net.Listener, string, error) {
	if !strings.HasPrefix(constant.DefaultGRPCAddress, constant.UnixPrefix) {
		return nil, "", errors.Errorf("listen address %s not supported", constant.DefaultGRPCAddress)
	}

	path := strings.TrimPrefix(constant.DefaultGRPCAddress, constant.UnixPrefix)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, "", err
	}

	l, err := sys.GetLocalListener(path, 0, 0)
	if err != nil {
		logrus.Errorf("Listen at unix address %s failed: %v", path, err)
		return nil, "", err
	}

	if err = os.Chmod(path, constant.DefaultGroupFileMode); err != nil {
		logrus.Errorf("Chmod for %s failed: %v", path, err)
		return nil, "", err
	}

	if err = util.ChangeGroup(path, group); err != nil {
		logrus.Errorf("Chown for %s failed: %v", path, err)
		return nil, "", err
	}

	return l, path, nil
}
