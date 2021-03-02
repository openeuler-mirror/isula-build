// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Zekun Liu
// Create: 2020-03-20
// Description: dockerfile parse related functions tests

package dockerfile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"isula.org/isula-build/pkg/parser"
)

func TestPreProcess(t *testing.T) {
	type testcase struct {
		name   string
		expect int
	}
	var testcases = []testcase{
		{
			name:   "busybox",
			expect: 7,
		},
		{
			name:   "busybox_with_directive",
			expect: 3,
		},
		{
			name:   "busybox_with_complex_line",
			expect: 8,
		},
		{
			name:   "busybox_with_space_line",
			expect: 3,
		},
		{
			name:   "busybox_with_commend_between",
			expect: 9,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "preprocess", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()
			lines := preProcess(r)
			assert.Equal(t, len(lines), tc.expect)
		})
	}
}

func TestFormat(t *testing.T) {
	type testcase struct {
		name    string
		expect  int
		wantErr bool
	}
	var testcases = []testcase{
		{
			name:   "busybox",
			expect: 7,
		},
		{
			name:   "busybox_with_directive",
			expect: 3,
		},
		{
			name:   "busybox_with_complex_line",
			expect: 4,
		},
		{
			name:   "yum_config",
			expect: 8,
		},
		{
			name:    "run_with_directive",
			wantErr: true,
		},
		{
			name:    "run_with_directive_with_space",
			wantErr: true,
		},
		{
			name:    "cmd_with_directive",
			wantErr: true,
		},
		{
			name:    "cmd_with_directive_with_space",
			wantErr: true,
		},
		{
			name:    "entrypoint_with_directive",
			wantErr: true,
		},
		{
			name:    "entrypoint_with_directive_with_space",
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "preprocess", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()
			var buf bytes.Buffer
			tee := io.TeeReader(r, &buf)
			rows := preProcess(tee)
			d, err := newDirective(bytes.NewReader(buf.Bytes()))
			assert.NilError(t, err)
			lines, err := format(rows, d)
			if (err != nil) != tc.wantErr {
				t.Errorf("Testing failed. Expected: %v, got: %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				assert.NilError(t, err, file)
				assert.Equal(t, len(lines), tc.expect)
			}
		})
	}
}

func TestFormatWithSpacesAfterEscapeToken(t *testing.T) {
	type testcase struct {
		name   string
		expect []int
	}
	var testcases = []testcase{
		{
			name:   "busybox_line_with_spaces",
			expect: []int{12, 20, 96, 87, 10},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "preprocess", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()
			var buf bytes.Buffer
			tee := io.TeeReader(r, &buf)
			rows := preProcess(tee)
			d, err := newDirective(bytes.NewReader(buf.Bytes()))
			assert.NilError(t, err)
			lines, err := format(rows, d)
			assert.NilError(t, err)
			for i, v := range tc.expect {
				assert.Equal(t, 1+len(lines[i].Command+lines[i].Raw), v)
			}

		})
	}
}

func TestParse(t *testing.T) {
	type testcase struct {
		name   string
		isErr  bool
		errStr string
	}
	var testcases = []testcase{
		{
			name: "busybox",
		},
		{
			name: "add_and_copy",
		},
		{
			name: "busybox_with_directive",
		},
		{
			name: "busybox_with_complex_line",
		},
		{
			name: "onbuild",
		},
		{
			name: "busybox_with_empty_continues_line",
		},
		{
			name:   "busybox_with_no_from",
			isErr:  true,
			errStr: "before FROM is not supported",
		},
		{
			name:   "busybox_with_empty_content",
			isErr:  true,
			errStr: "no instructions in Dockerfile",
		},
		{
			name: "busybox_no_command",
		},
		{
			name:   "env_before_from",
			isErr:  true,
			errStr: "command ENV before FROM is not supported",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "preprocess", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()

			df := dockerfile{}
			_, err = df.Parse(r, false)

			if !tc.isErr {
				assert.NilError(t, err, file)
			} else {
				assert.ErrorContains(t, err, tc.errStr)
			}
		})
	}
}

func TestParseContainSingleFrom(t *testing.T) {
	testcases := []struct {
		name      string
		isErr     bool
		committed bool
	}{
		{
			name:      "busybox_with_from_only",
			isErr:     false,
			committed: false,
		}, {
			name:      "busybox_ubuntu_centos",
			isErr:     false,
			committed: false,
		}, {
			name:      "compelte_stage_with_single_from_stage",
			isErr:     false,
			committed: false,
		}, {
			name:      "single_from_stage_with_complete_stage",
			isErr:     false,
			committed: true,
		}, {
			name:      "final_single_from_stage_depend_on_previous_stage",
			isErr:     false,
			committed: true,
		}, {
			name:      "final_stage_depend_on_previous_stage",
			isErr:     false,
			committed: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("testfiles", "preprocess", tc.name)
			r, err := os.Open(file)
			assert.NilError(t, err)
			defer r.Close()

			df := dockerfile{}
			playbook := &parser.PlayBook{}
			playbook, err = df.Parse(r, false)

			if !tc.isErr {
				assert.NilError(t, err, file)
				if tc.committed {
					needCommit := false
					for _, page := range playbook.Pages {
						needCommit = page.NeedCommit || needCommit
					}
					assert.Equal(t, needCommit, true)
				}
			}
		})
	}
}

func TestParseIgnore(t *testing.T) {
	dockerignore := `
# comment
*/temp*
*/*/temp*
temp?`
	ctxDir := fs.NewDir(t, t.Name(), fs.WithFile(ignoreFile, dockerignore))
	defer ctxDir.Remove()

	df := dockerfile{}
	ignores, err := df.ParseIgnore(ctxDir.Path())
	assert.NilError(t, err)

	expected := strings.Split(dockerignore, "\n")
	// expected[2:0]: trim empty line and comment line
	assert.DeepEqual(t, ignores, expected[2:])
}

func TestParseIgnoreWithNoFile(t *testing.T) {
	ctxDir := fs.NewDir(t, t.Name())
	defer ctxDir.Remove()

	df := dockerfile{}
	ignores, err := df.ParseIgnore(ctxDir.Path())
	assert.NilError(t, err)
	assert.DeepEqual(t, ignores, []string{})
}

func TestParseIgnoreWithDir(t *testing.T) {
	ctxDir := fs.NewDir(t, t.Name(), fs.WithDir(ignoreFile))
	defer ctxDir.Remove()

	df := dockerfile{}
	_, err := df.ParseIgnore(ctxDir.Path())
	assert.ErrorContains(t, err, "a directory")
}

func TestParseWithHeadingArgs(t *testing.T) {
	content := `
ARG testArg
ARG testArg2
ARG testArg3=val
ARG testArg=val
FROM alpine AS uuid
RUN ls
`
	df := dockerfile{}
	playbook, err := df.Parse(bytes.NewReader([]byte(content)), false)
	assert.NilError(t, err)

	expectedArgs := []string{"testArg", "testArg2", "testArg3=val", "testArg=val"}
	assert.DeepEqual(t, playbook.HeadingArgs, expectedArgs)
}

func TestParseWithWrongHeadingCmd(t *testing.T) {
	content := `
ARG testArg
ENV testArg2 val
FROM alpine AS uuid
`
	df := dockerfile{}
	_, err := df.Parse(bytes.NewReader([]byte(content)), false)
	assert.ErrorContains(t, err, "before FROM is not supported")
}

func TestParseWithStageName(t *testing.T) {
	content := `
ARG testArg
FROM alpine AS uuid
COPY uuid /src/data

FROM alpine AS date
COPY date /src/data

FROM alpine
COPY --from=uuid --chown=test /src/data /uuid
COPY --from=date /src/data /date
ADD uuid --chown=55:mygroup /src
`
	df := dockerfile{}
	playbook, err := df.Parse(bytes.NewReader([]byte(content)), false)
	assert.NilError(t, err)
	assert.Equal(t, playbook.Pages[0].Name, "uuid")
	assert.Equal(t, playbook.Pages[1].Name, "date")
	assert.Equal(t, playbook.Pages[2].Name, "2")
}

func TestGetPageName(t *testing.T) {
	type testcase struct {
		name      string
		line      parser.Line
		isErr     bool
		errStr    string
		expectStr string
	}
	var testcases = []testcase{
		{
			name: "normal from line",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
				},
			},
			expectStr: "0",
		},
		{
			name: "with as from line",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
					{Value: "as"},
					{Value: "isula"},
				},
			},
			expectStr: "isula",
		},
		{
			name: "with illegal as name",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
					{Value: "as"},
					{Value: "!@#!@F!#$T!%!@$#"},
				},
			},
			isErr: true,
		},
		{
			name: "with number as name",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
					{Value: "as"},
					{Value: "1isula"},
				},
			},
			expectStr: "1isula",
		},
		{
			name: "name with 64",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
					{Value: "as"},
					{Value: "0123456789012345678901234567890123456789012345678901234567890123"},
				},
			},
			expectStr: "0123456789012345678901234567890123456789012345678901234567890123",
		},
		{
			name: "name with 64",
			line: parser.Line{
				Command: "FROM",
				Cells: []*parser.Cell{
					{Value: "alpine"},
					{Value: "as"},
					{Value: "01234567890123456789012345678901234567890123456789012345678901234"},
				},
			},
			isErr: true,
		},
	}
	for i, tc := range testcases {
		name, err := getPageName(&tc.line, i)
		assert.Equal(t, err != nil, tc.isErr, tc.name)
		if err != nil {
			assert.ErrorContains(t, err, "invalid page name")
		}
		if err == nil {
			assert.Equal(t, name, tc.expectStr, tc.name)
		}
	}
}

func TestParseWithFuzzCorpus(t *testing.T) {
	var testcases = []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "fuzz 1",
			data:    `COPY --from`,
			wantErr: true,
		},
		{
			name:    "fuzz 2",
			data:    `ADD --chown`,
			wantErr: true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parser.NewParser(parser.DefaultParser)
			assert.NilError(t, err)

			_, err = p.Parse(bytes.NewBufferString(tt.data), false)
			if (err != nil) != tt.wantErr {
				t.Errorf("Testing failed. Expected: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestParserParseOnBuild(t *testing.T) {
	type testcase struct {
		name       string
		dockerfile string
		isErr      bool
		errStr     string
	}
	var testcases = []testcase{
		{
			name:       "one line onbuild",
			dockerfile: "RUN mkdir /tmp",
			isErr:      false,
		},
		{
			name:       "multi-line onbuild",
			dockerfile: "ADD . /app/src\n RUN /usr/local/bin/python-build --dir /app/src",
			isErr:      false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			df := dockerfile{}
			_, err := df.Parse(bytes.NewReader([]byte(tc.dockerfile)), true)
			if !tc.isErr {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errStr)
			}
		})
	}
}
