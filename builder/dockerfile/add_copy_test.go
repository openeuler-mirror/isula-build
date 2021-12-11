// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zhongkai Lei, Feiyu Yang
// Create: 2020-03-20
// Description: ADD and COPY command related functions tests

package dockerfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/containers/storage/pkg/idtools"
	securejoin "github.com/cyphar/filepath-securejoin"
	"gotest.tools/v3/assert"

	constant "isula.org/isula-build"
	"isula.org/isula-build/util"
)

type testEnvironment struct {
	contextDir string
	workDir    string
	mountpoint string
}

func prepareResolveDestEnvironment(prepareCmd, workdir, mountpoint string) (*testEnvironment, string) {
	testDir, _ := ioutil.TempDir("", "test_add_copy")

	realMountpoint := filepath.Join(testDir, "mountpoint")
	os.MkdirAll(realMountpoint, constant.DefaultRootDirMode)
	if mountpoint != realMountpoint {
		realMountpoint, _ = securejoin.SecureJoin("", mountpoint)
	}

	contextDir := filepath.Join(testDir, "contextDir")

	os.Chdir(testDir)
	if prepareCmd != "" {
		cmd := exec.Command("/bin/sh", "-c", prepareCmd)
		err := cmd.Run()
		fmt.Println(err)
	}

	realWorkDir := filepath.Join(testDir, mountpoint, workdir)
	os.MkdirAll(realWorkDir, constant.DefaultRootDirMode)
	os.Chdir(realWorkDir)

	return &testEnvironment{
		contextDir: contextDir,
		workDir:    workdir,
		mountpoint: realMountpoint,
	}, testDir
}

func prepareResolveSourceEnvironment(prepareCmd, contextDir string) {
	os.Chdir(contextDir)
	if prepareCmd != "" {
		cmd := exec.Command("/bin/sh", "-c", prepareCmd)
		cmd.Run()
	}
}

func TestResolveCopyDest(t *testing.T) {
	type args struct {
		rawDest    string
		workDir    string
		mountpoint string
	}
	tests := []struct {
		name       string
		args       args
		want       string
		wantErr    bool
		prepareCmd string
	}{
		{
			args: args{
				rawDest:    "foo/bar",
				workDir:    "/workdir",
				mountpoint: "mountpoint",
			},
		},
		{
			args: args{
				rawDest:    "foo/bar",
				workDir:    "",
				mountpoint: "mountpoint",
			},
		},
		{
			args: args{
				rawDest:    "foo/bar",
				workDir:    ".",
				mountpoint: "mountpoint",
			},
		},
		{
			args: args{
				rawDest:    "foo/bar/",
				workDir:    ".",
				mountpoint: "mountpoint",
			},
		},
		{
			args: args{
				rawDest:    ".",
				workDir:    "/foo",
				mountpoint: "mountpoint",
			},
		},
		{
			args: args{
				rawDest:    "foo/bar",
				workDir:    ".",
				mountpoint: "mountPointSoft",
			},
			prepareCmd: "ln -sf mountpoint mountPointSoft",
		},
	}
	for _, tt := range tests {
		testEnvironment, testDir := prepareResolveDestEnvironment(tt.prepareCmd, tt.args.workDir, tt.args.mountpoint)
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCopyDest(tt.args.rawDest, testEnvironment.workDir, testEnvironment.mountpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveCopyDest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want = filepath.Join(testEnvironment.mountpoint, testEnvironment.workDir, tt.args.rawDest)
			if tt.args.rawDest[len(tt.args.rawDest)-1] == os.PathSeparator {
				tt.want += string(os.PathSeparator)
			}
			if got != tt.want {
				t.Errorf("resolveCopyDest() got = %v, want %v", got, tt.want)
			}
		})
		os.RemoveAll(testDir)
	}
}

func TestResolveCopySource(t *testing.T) {
	testDir := "/tmp/TestResolveCopySource/"
	testContextDir := filepath.Join(testDir, "testContextDir")
	type args struct {
		isAdd      bool
		rawSources []string
		dest       string
	}
	tests := []struct {
		name       string
		args       args
		prepareCmd string
		want       copyDetails
		wantErr    bool
	}{
		{
			args: args{
				isAdd:      true,
				rawSources: []string{"http://foo/bar", "https://foo/bar"},
				dest:       "/foo/bar",
			},
			want: copyDetails{
				"/foo/bar": []string{"http://foo/bar", "https://foo/bar"},
			},
		},
		{
			args: args{
				isAdd:      false,
				rawSources: []string{"http://foo/bar", "https://foo/bar"},
				dest:       "/foo/bar",
			},
			wantErr: true,
		},
		{
			args: args{
				isAdd:      true,
				rawSources: []string{"foo", "bar"},
				dest:       "/foo/bar/",
			},
			prepareCmd: "touch foo && touch bar",
			want: copyDetails{
				"/foo/bar/": []string{filepath.Join(testContextDir, "foo"), filepath.Join(testContextDir, "bar")},
			},
			wantErr: false,
		},
		{
			args: args{
				isAdd:      true,
				rawSources: []string{"foo", "bar"},
				dest:       "/foo/bar/",
			},
			prepareCmd: "touch foo && ln -sf foo bar",
			want: copyDetails{
				"/foo/bar/":    []string{filepath.Join(testContextDir, "foo")},
				"/foo/bar/bar": []string{filepath.Join(testContextDir, "foo")},
			},
			wantErr: false,
		},
		{
			args: args{
				isAdd:      true,
				rawSources: []string{"foo", "bar"},
				dest:       "/foo/bar",
			},
			prepareCmd: "touch foo && ln -sf foo bar",
			want: copyDetails{
				"/foo/bar": []string{filepath.Join(testContextDir, "foo"), filepath.Join(testContextDir, "foo")},
			},
			wantErr: false,
		},
		{
			args: args{
				isAdd:      true,
				rawSources: []string{"bar"},
				dest:       "/foo/bar/",
			},
			prepareCmd: "touch ../foo && ln -sf ../foo bar",
			want: copyDetails{
				"/foo/bar/bar": []string{filepath.Join(testContextDir, "foo")},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		err := os.MkdirAll(testContextDir, constant.DefaultRootDirMode)
		assert.NilError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			prepareResolveSourceEnvironment(tt.prepareCmd, testContextDir)
			got, err := resolveCopySource(tt.args.isAdd, tt.args.rawSources, tt.args.dest, testContextDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveCopySource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveCopySource() got = %v, want %v", got, tt.want)
			}
		})
		os.RemoveAll(testDir)
	}
}

func TestAddFile(t *testing.T) {
	realSrc := fmt.Sprintf("/tmp/test-%d", util.GenRandInt64())
	dest := fmt.Sprintf("/tmp/test2-%d", util.GenRandInt64())
	err := exec.Command("/bin/sh", "-c", "touch "+realSrc).Run()
	assert.NilError(t, err)

	opt := &addOptions{}

	err = addFile(realSrc, realSrc, dest, opt)
	assert.NilError(t, err)
	_, err = os.Stat(dest)
	assert.NilError(t, err)
	err = os.Remove(realSrc)
	assert.NilError(t, err)
	err = os.Remove(dest)
	assert.NilError(t, err)

	tarFile := fmt.Sprintf("/tmp/a-%d.tar.gz", util.GenRandInt64())
	srcFile1 := fmt.Sprintf("/tmp/test-%d", util.GenRandInt64())
	srcFile2 := fmt.Sprintf("/tmp/test2-%d", util.GenRandInt64())
	err = exec.Command("/bin/sh", "-c", "touch "+srcFile1+" "+srcFile2+
		" && tar -czf "+tarFile+" "+srcFile1+" "+srcFile2).Run()
	assert.NilError(t, err)
	opt.extract = true
	err = addFile(tarFile, tarFile, dest, opt)
	assert.NilError(t, err)

	fi, err := os.Stat(dest)
	assert.NilError(t, err)
	assert.Equal(t, fi.IsDir(), true)
	fi, err = os.Stat(dest + srcFile1)
	assert.NilError(t, err)
	assert.Equal(t, fi.Name(), filepath.Base(srcFile1))

	err = os.RemoveAll(dest)
	assert.NilError(t, err)
	err = os.Remove(srcFile1)
	assert.NilError(t, err)
	err = os.Remove(srcFile2)
	assert.NilError(t, err)
	err = os.Remove(tarFile)
	assert.NilError(t, err)
}

func TestAdd(t *testing.T) {
	ignores := []string{"a", "b"}
	contextDir := fmt.Sprintf("/tmp/context-%d", util.GenRandInt64())
	contextDir2 := fmt.Sprintf("/tmp/context-%d", util.GenRandInt64())
	matcher, err := util.GetIgnorePatternMatcher(ignores, contextDir, "")
	assert.NilError(t, err)

	file1 := contextDir + "/a"
	file2 := contextDir + "/b"
	dir := contextDir + "/dir"
	file3 := dir + "/c"
	err = exec.Command("/bin/sh", "-c", "mkdir -p "+contextDir+" && touch "+file1+" "+file2).Run()
	assert.NilError(t, err)
	err = exec.Command("/bin/sh", "-c", "mkdir -p "+dir+" && touch "+file3).Run()
	assert.NilError(t, err)

	c := cmdBuilder{}
	opt := &addOptions{
		matcher:   matcher,
		chownPair: idtools.IDPair{UID: 1000, GID: 1001},
		extract:   true,
	}
	err = c.add(contextDir+"/*", contextDir2+"/", opt)
	assert.NilError(t, err)

	_, err = os.Stat(contextDir2 + "/" + filepath.Base(file1))
	assert.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(contextDir2 + "/" + filepath.Base(file2))
	assert.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(contextDir2 + "/" + filepath.Base(dir))
	assert.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(contextDir2 + "/" + filepath.Base(file3))
	assert.NilError(t, err)

	err = os.RemoveAll(contextDir)
	assert.NilError(t, err)
	err = os.RemoveAll(contextDir2)
	assert.NilError(t, err)
}
