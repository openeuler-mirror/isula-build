// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Jingxiao Lu
// Create: 2020-07-24
// Description: Test cases for Builder package

package builder

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	constant "isula.org/isula-build"
	pb "isula.org/isula-build/api/services"
	"isula.org/isula-build/builder/dockerfile"
	"isula.org/isula-build/store"
	"isula.org/isula-build/util"
)

var (
	localStore store.Store
	rootDir    = "/tmp/isula-build/builder"
)

func init() {
	reexec.Init()
	dataRoot := rootDir + "/data"
	runRoot := rootDir + "/run"
	store.SetDefaultStoreOptions(store.DaemonStoreOptions{
		DataRoot: dataRoot,
		RunRoot:  runRoot,
	})
	localStore, _ = store.GetStore()
}

func clean() {
	if err := unix.Unmount(rootDir+"/data/overlay", 0); err != nil {
		fmt.Printf("umount dir %s failed: %v\n", rootDir+"/data/overlay", err)
	}

	if err := os.RemoveAll(rootDir); err != nil {
		fmt.Printf("remove test root dir %s failed: %v\n", rootDir, err)
	}
}

func TestNewBuilder(t *testing.T) {
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	key, err := util.GenerateRSAKey(util.DefaultRSAKeySize)
	assert.NilError(t, err)

	type args struct {
		ctx         context.Context
		store       *store.Store
		req         *pb.BuildRequest
		runtimePath string
		buildDir    string
		runDir      string
	}
	tests := []struct {
		name    string
		args    args
		want    Builder
		wantErr bool
	}{
		{
			name: "ctr-img docker",
			args: args{
				ctx:      context.Background(),
				store:    &localStore,
				req:      &pb.BuildRequest{BuildType: constant.BuildContainerImageType, Format: "docker"},
				buildDir: tmpDir.Path(),
				runDir:   tmpDir.Path(),
			},
			want:    &dockerfile.Builder{},
			wantErr: false,
		},
		{
			name: "ctr-img oci",
			args: args{
				ctx:      context.Background(),
				store:    &localStore,
				req:      &pb.BuildRequest{BuildType: constant.BuildContainerImageType, Format: "oci"},
				buildDir: tmpDir.Path(),
				runDir:   tmpDir.Path(),
			},
			want:    &dockerfile.Builder{},
			wantErr: false,
		},
		{
			name: "unsupported type",
			args: args{
				ctx:   context.Background(),
				store: &localStore,
				req:   &pb.BuildRequest{BuildType: "Unknown", Format: "docker"},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBuilder(tt.args.ctx, tt.args.store, tt.args.req, tt.args.runtimePath, tt.args.buildDir, tt.args.runDir, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBuilder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewBuilder() got = %v, want %v", reflect.TypeOf(got), reflect.TypeOf(tt.want))
			}
		})
	}
	clean()
}
