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
// Description: dockerfile parse related functions

package dockerfile

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	constant "isula.org/isula-build"
	"isula.org/isula-build/pkg/parser"
	"isula.org/isula-build/util"
)

var (
	regPageName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_\\.]{0,63}$`)
	regComment  = regexp.MustCompile(`^(\s*|)#.*$`)
)

type rowLine struct {
	lineNum int
	content string
}

type dockerfile struct {
}

// Parse the given Dockerfile and return a PlayBook
func (df *dockerfile) Parse(r io.Reader, onbuild bool) (*parser.PlayBook, error) {
	var (
		buf bytes.Buffer
		d   *directive
		err error
	)
	tee := io.TeeReader(r, &buf)

	// 1. scan each line, trim comment and space line and load to a rowLine
	rowLines := preProcess(tee)

	// 2. init directive and it need to behand the full scan after preprocess
	d, err = newDirective(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}

	// 3. along with directive, process continue char and init each parser.Line
	lines, err := format(rowLines, d)
	if err != nil {
		return nil, err
	}

	// 4. according to the command handler, update each parse.Line
	for _, line := range lines {
		command := line.Command
		if _, ok := preHandlers[command]; !ok {
			return nil, errors.Errorf("command %s is invalid", command)
		}
		if err = preHandlers[command](line); err != nil {
			return nil, err
		}
	}

	// 5. truncate ARGs before the first FROM if there is
	headingArgs, err := truncHeadingArgs(&lines, onbuild)
	if err != nil {
		return nil, err
	}

	// 6. construct the page with lines
	pages, err := constructPages(lines, onbuild)
	if err != nil {
		return nil, err
	}

	// 7. construct the playbook with pages
	playbook := &parser.PlayBook{
		HeadingArgs: headingArgs,
		Pages:       pages,
	}
	return playbook, nil
}

func newRowLine(num int, content string) *rowLine {
	return &rowLine{
		lineNum: num,
		content: content,
	}
}

// preprocess the Dockerfile and get the effective physical line
func preProcess(r io.Reader) []*rowLine {
	rowLines := make([]*rowLine, 0, 0)
	scanner := bufio.NewScanner(r)
	lineNum := 1
	for scanner.Scan() {
		bytes := regComment.ReplaceAll(scanner.Bytes(), []byte{})
		if len(bytes) != 0 && strings.TrimSpace(string(bytes)) != "" {
			r := newRowLine(lineNum, string(bytes))
			rowLines = append(rowLines, r)
		}
		lineNum++
	}

	return rowLines
}

// trim continue char and format it into parser.Line
func format(rows []*rowLine, d *directive) ([]*parser.Line, error) {
	lines := make([]*parser.Line, 0, len(rows))
	for i := 0; i <= len(rows)-1; i++ {
		if rows[i] == nil {
			continue
		}
		text := rows[i].content
		line := &parser.Line{
			Begin: rows[i].lineNum,
			Flags: make(map[string]string, 0),
		}

		var logicLine string
		trimLine := strings.TrimSpace(text)
		// concat continues lines until no escapeToken at the end of the line
		for len(trimLine) != 0 && trimLine[len(trimLine)-1] == d.escapeToken {
			if str := trimLine[:len(trimLine)-1]; len(str) != 0 {
				logicLine += str
			}
			i++
			if i > len(rows)-1 {
				break
			}
			text = rows[i].content
			trimLine = strings.TrimSpace(text)
		}

		if i <= len(rows)-1 {
			logicLine += trimLine
			line.End = rows[i].lineNum
		} else {
			line.End = rows[i-1].lineNum
		}
		fields := strings.SplitN(logicLine, " ", 2)
		const validLineLen = 2
		// we do not allow empty raw command been passed
		if len(fields) < validLineLen || len(fields[1]) == 0 {
			return nil, errors.Errorf("line %q should have at least two fields", logicLine)
		}
		line.Command = strings.ToUpper(fields[0])
		line.Raw = strings.TrimSpace(fields[1])
		lines = append(lines, line)
	}

	return lines, nil
}

func getPageName(line *parser.Line, pageNum int) (string, error) {
	name := strconv.Itoa(pageNum)
	// FROM euleros:latest AS euleros
	const setNameFromLen, nameIndex = 3, 2
	if len(line.Cells) == setNameFromLen {
		name = line.Cells[nameIndex].Value
	}
	if !regPageName.MatchString(name) {
		return "", errors.Errorf("invalid page name: %q, can't start with illegal symbols or length beyond 64", name)
	}

	return name, nil
}

func constructPages(lines []*parser.Line, onbuild bool) ([]*parser.Page, error) {
	if len(lines) == 0 {
		return nil, errors.New("no instructions in Dockerfile")
	}

	var (
		pageMap     = make(map[string]*parser.Page)
		pages       = make([]*parser.Page, 0, 0)
		currentPage *parser.Page
		pageNum     int
	)

	for _, line := range lines {
		if line == nil {
			continue
		}
		if onbuild && currentPage == nil {
			currentPage = &parser.Page{
				Lines: make([]*parser.Line, 0, 0),
				Begin: line.Begin,
				End:   line.End,
			}
		}
		if line.Command == From {
			if onbuild {
				return nil, errors.New("onbuild does not support the from command")
			}
			if currentPage != nil {
				pages = append(pages, currentPage)
			}

			name, err := getPageName(line, pageNum)
			if err != nil {
				return nil, err
			}
			pageNum++
			// new a page used for the next stage
			page := &parser.Page{
				Name:  name,
				Begin: line.Begin,
				End:   line.End,
				Lines: make([]*parser.Line, 0, 0),
			}
			// page name comes from the last cell from "FROM {image} AS {name}
			// or named it with the index of stage in this dockerfile
			const tokenNumFromAsName = 3
			if len(line.Cells) == tokenNumFromAsName {
				page.Name = line.Cells[tokenNumFromAsName-1].Value
			} else {
				page.Name = strconv.Itoa(len(pages))
			}
			pageMap[page.Name] = page
			// if the base image for current stage is from the previous stage,
			// mark the previous stage need to commit, for only from command we don't commit
			if from, ok := pageMap[line.Cells[0].Value]; ok && len(from.Lines) > 1 {
				from.NeedCommit = true
			}
			currentPage = page
		}
		// because a valid dockerfile is always start with 'FROM' command here, so no need
		// to check whether currentPage is nil or not
		currentPage.End = line.End
		currentPage.AddLine(line)
	}
	// the last stage always need to commit except page that contains only from command
	if len(currentPage.Lines) > 1 {
		currentPage.NeedCommit = true
	}
	pages = append(pages, currentPage)

	if len(pages) == 0 {
		return nil, errors.New("no stages in Dockerfile")
	}

	return pages, nil
}

// truncHeadingArgs Handle those ARGs before first FROM in the file
// returns the truncated lines and converted heading args
func truncHeadingArgs(lines *[]*parser.Line, onbuild bool) ([]string, error) {
	args := make([]string, 0, 0)
	if onbuild {
		return args, nil
	}

	var num int
	for _, line := range *lines {
		if line.Command == From {
			// meets the first FROM
			break
		}
		// only ARG command can be put before the first FROM
		if line.Command != Arg {
			return nil, errors.Errorf("command %s before FROM is not supported", line.Command)
		}

		args = append(args, line.Cells[0].Value)
		num++
	}
	*lines = (*lines)[num:]

	return args, nil
}

const ignoreFile = ".dockerignore"

// ParseIgnore parses the .dockerignore file in the provide dir, which
// must be the context directory
func (df *dockerfile) ParseIgnore(dir string) ([]string, error) {
	var ignores = make([]string, 0, 0)

	fullPath := path.Join(dir, ignoreFile)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ignores, nil
		}
		return ignores, errors.Wrap(err, "state dockerignore file failed")
	}
	if err := util.CheckFileSize(fullPath, constant.MaxFileSize); err != nil {
		return ignores, err
	}

	// file exists and it is a real file
	f, err := os.Open(filepath.Clean(fullPath))
	if err != nil {
		return ignores, errors.Wrap(err, "open dockerignore file failed")
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			logrus.Warningf("Closing fd on dockerignore file failed: %v", cerr)
		}
	}()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		// ignore empty lines and lines starting with '#' (which are comments)
		if len(line) == 0 || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		ignores = append(ignores, line)
	}

	return ignores, nil
}
