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
// Description: This file stores functions which used for grpc backend processing

package daemon

import (
	"sync"

	"google.golang.org/grpc"

	pb "isula.org/isula-build/api/services"
)

// Backend lives in Server, handles GRPC requests from Client
type Backend struct {
	sync.RWMutex
	daemon *Daemon
	status map[string]*status
}

// NewBackend create an instance of backend
func (d *Daemon) NewBackend() {
	d.backend = &Backend{
		daemon: d,
		status: make(map[string]*status),
	}
}

// Register registers the services belong to backend
func (b *Backend) Register(s *grpc.Server) {
	pb.RegisterControlServer(s, b)
}

func (b *Backend) deleteStatus(buildID string) {
	b.Lock()
	delete(b.status, buildID)
	b.Unlock()
}

func (b *Backend) deleteAllStatus() {
	b.Lock()
	b.status = make(map[string]*status)
	b.Unlock()
}
