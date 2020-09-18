// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-01-20
// Description: This file is used for grpc client setting

package main

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
)

const (
	defaultStartTimeout = 20 * time.Second
	minStartTimeout     = 100 * time.Millisecond
	maxStartTimeout     = 120 * time.Second
	defaultGrpcMaxDelay = 3 * time.Second
)

// Cli defines grpc client
type Cli interface {
	Close() error
	Client() pb.ControlClient
}

// GrpcClient lives in Client, sends GRPC requests to Server
type GrpcClient struct {
	conn   *grpc.ClientConn
	client pb.ControlClient
}

// NewClient returns an instance of grpc client
func NewClient(ctx context.Context) (*GrpcClient, error) {
	bc := backoff.DefaultConfig
	bc.MaxDelay = defaultGrpcMaxDelay
	connParams := grpc.ConnectParams{Backoff: bc}
	gopts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithConnectParams(connParams),
		grpc.WithContextDialer(dialerCtx(ctx, "unix", strings.TrimPrefix(constant.DefaultGRPCAddress, constant.UnixPrefix))),
	}

	if !isSocketReady() {
		return nil, errors.Errorf("invalid socket path: %s", constant.DefaultGRPCAddress)
	}

	conn, err := grpc.DialContext(ctx, constant.DefaultGRPCAddress, gopts...)
	if err != nil {
		return nil, err
	}

	cli := &GrpcClient{
		conn:   conn,
		client: pb.NewControlClient(conn),
	}

	startTimeout, err := getStartTimeout(cliOpts.Timeout)
	if err != nil {
		return nil, err
	}

	healthCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()
	connected, err := cli.HealthCheck(healthCtx)
	if !connected || err != nil {
		return nil, errors.Wrapf(err, "Cannot connect to the isula-builder at %s. Is the isula-builder running?\nError", constant.DefaultGRPCAddress)
	}

	return cli, nil
}

// Close close grpc connection
func (c *GrpcClient) Close() error {
	return c.conn.Close()
}

// Client returns grpc client
func (c *GrpcClient) Client() pb.ControlClient {
	return c.client
}

func getStartTimeout(timeout string) (time.Duration, error) {
	if timeout == "" {
		return defaultStartTimeout, nil
	}
	timeParse, err := time.ParseDuration(timeout)
	if err != nil {
		return -1, err
	}
	if timeParse < minStartTimeout || timeParse > maxStartTimeout {
		return -1, errors.Errorf("invalid timeout value: %s, supported range [%s, %s]", timeout, minStartTimeout, maxStartTimeout)
	}
	return timeParse, nil
}

func isSocketReady() bool {
	path := strings.TrimPrefix(constant.DefaultGRPCAddress, constant.UnixPrefix)
	info, err := os.Stat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return false
	}
	return true
}

func dialerCtx(ctx context.Context, socket, address string) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		var d net.Dialer
		newCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		return d.DialContext(newCtx, socket, address)
	}
}

// HealthCheck checks whether daemon is running within timeout
func (c *GrpcClient) HealthCheck(ctx context.Context) (bool, error) {
	res, err := c.client.HealthCheck(ctx, &types.Empty{})
	if err == nil {
		return res.GetStatus() == pb.HealthCheckResponse_SERVING, nil
	}
	return false, err
}
