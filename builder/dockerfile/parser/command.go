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
// Create: 2020-03-20
// Description: command parse related functions

// Package dockerfile is used to parse dockerfile
package dockerfile

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	"isula.org/isula-build/pkg/parser"
)

const (
	// Add is a Dockerfile ADD command
	Add = "ADD"
	// Arg is a Dockerfile ARG command
	Arg = "ARG"
	// Cmd is a Dockerfile CMD command
	Cmd = "CMD"
	// Copy is a Dockerfile COPY command
	Copy = "COPY"
	// Entrypoint is a Dockerfile ENTRYPOINT command
	Entrypoint = "ENTRYPOINT"
	// Env is a Dockerfile ENV command
	Env = "ENV"
	// Expose is a Dockerfile EXPOSE command
	Expose = "EXPOSE"
	// From is a Dockerfile FROM command
	From = "FROM"
	// Healthcheck is a Dockerfile HEALTHCHECK command
	Healthcheck = "HEALTHCHECK"
	// Label is a Dockerfile LABEL command
	Label = "LABEL"
	// Maintainer is a  Dockerfile MAINTAINER command
	Maintainer = "MAINTAINER"
	// OnBuild is a Dockerfile ONBUILD command
	OnBuild = "ONBUILD"
	// Run is a Dockerfile RUN command
	Run = "RUN"
	// Shell is a Dockerfile SHELL command
	Shell = "SHELL"
	// StopSignal is a Dockerfile STOPSIGNAL command
	StopSignal = "STOPSIGNAL"
	// User is a Dockerfile USER command
	User = "USER"
	// Volume is a Dockerfile VOLUME command
	Volume = "VOLUME"
	// WorkDir is a Dockerfile WORKDIR command
	WorkDir = "WORKDIR"
)

const (
	// HealthCheckStartPeriod is a "start-period" Flag for HealthCheck
	HealthCheckStartPeriod = "start-period"
	// HealthCheckInterval is a "interval" Flag for HealthCheck
	HealthCheckInterval = "interval"
	// HealthCheckTimeout is a "timeout" Flag for HealthCheck
	HealthCheckTimeout = "timeout"
	// HealthCheckRetries is a "retries" Flag for HealthCheck
	HealthCheckRetries = "retries"
)

var (
	// <key> [=value], and key can be any string which not contain space and '='
	regKeyOrKeyValue = regexp.MustCompile(`^([^\s=]+)(|=[^\s]+)$`)
	// key value, and value can start with '='
	regKeyValue = regexp.MustCompile(`^([\w-:\.\$\{\}\+]+)\s+([^\s].*)$`)
	// [value1,value2,...valueN]
	regJSONArray = regexp.MustCompile(`^\s*\[.*\]\s*$`)
	// chown flag value regexp
	regChownFlag = regexp.MustCompile(`^((\w+)|(\w+:\w+))$`)
	// --<flag>
	regCmdFlag = regexp.MustCompile(`^--\S+`)
	// cmd flags map
	cmdFlagRegs = map[string]map[string]*regexp.Regexp{
		Add: {
			"chown": regChownFlag,
		},
		Copy: {
			"chown": regChownFlag,
			"from":  nil,
		},
		Healthcheck: {
			HealthCheckStartPeriod: nil,
			HealthCheckInterval:    nil,
			HealthCheckTimeout:     nil,
			HealthCheckRetries:     nil,
		},
	}

	errJSONArrayIsNotString = errors.New("only string type is allowd as JSON format arrays")

	preHandlers map[string]func(line *parser.Line) error
)

func init() {
	preHandlers = map[string]func(line *parser.Line) error{
		Add:         parseAdd,
		Arg:         parseArg,
		Cmd:         parseCmd,
		Copy:        parseCopy,
		Entrypoint:  parseEntrypoint,
		Env:         parseEnv,
		Expose:      parseExpose,
		From:        parseFrom,
		Healthcheck: parseHealthCheck,
		Label:       parseLabel,
		Maintainer:  parseMaintainer,
		Run:         parseRun,
		Shell:       parseShell,
		OnBuild:     parseOnBuild,
		Volume:      parseVolume,
		StopSignal:  parseStopSignal,
		User:        parseUser,
		WorkDir:     parseWorkDir,
	}

	parser.Register("dockerfile", &dockerfile{})
}

func parseAdd(line *parser.Line) error {
	if err := parseAddCopyVolume(line, Add); err != nil {
		return err
	}

	// ADD has at least two arguments, e.g. ADD [--chown=<user>:<group>] <src>... <dest>
	if len(line.Cells) < 2 {
		return errors.New("parse failed, ADD requires at least two arguments")
	}

	return nil
}

func parseArg(line *parser.Line) error {
	return parseKeyOrKeyValue(line)
}

func parseCmd(line *parser.Line) error {
	return parseCmdEntrypointRun(line)
}

func parseCopy(line *parser.Line) error {
	if err := parseAddCopyVolume(line, Copy); err != nil {
		return err
	}

	// COPY has at least two arguments, e.g. COPY [--chown=<user>:<group>] <src>... <dest>
	if len(line.Cells) < 2 {
		return errors.New("parse failed, COPY requires at least two arguments")
	}

	return nil
}

func parseEntrypoint(line *parser.Line) error {
	return parseCmdEntrypointRun(line)
}

func parseEnv(line *parser.Line) error {
	return parseKeyValue(line)
}

func parseExpose(line *parser.Line) error {
	return parseMaybeString(line)
}

func parseFrom(line *parser.Line) error {
	return parseMaybeString(line)
}

func parseHealthCheck(line *parser.Line) error {
	lineWithoutCmdFlags, err := extractFlags(line, Healthcheck)
	if err != nil {
		return err
	}

	var reg = regexp.MustCompile(`(?i)(cmd|none)($|\s+)`)
	matches := reg.FindStringSubmatch(lineWithoutCmdFlags)
	if matches == nil {
		return errors.New("unknown type for healthcheck, need cmd or none")
	}
	match := matches[1]
	if strings.Index(lineWithoutCmdFlags, match) != 0 {
		return errors.New("wrong argument before CMD or NONE")
	}
	checkType := strings.ToUpper(match)
	if checkType == "CMD" && lineWithoutCmdFlags == match {
		return errors.New("missing command after healthcheck cmd")
	}
	if checkType == "NONE" && len(lineWithoutCmdFlags) > len(checkType) {
		return errors.New("none should not take arguments behind it")
	}

	cell := &parser.Cell{
		Value: checkType,
	}
	line.AddCell(cell)

	if checkType == "NONE" {
		return nil
	}

	rest := strings.TrimPrefix(lineWithoutCmdFlags, match)
	raw := line.Raw
	line.Raw = strings.TrimSpace(rest)
	line.Command = Cmd
	err = parseCmd(line)

	// recover
	line.Raw = raw
	line.Command = Healthcheck

	return err
}

func parseLabel(line *parser.Line) error {
	return parseKeyValue(line)
}

func parseMaintainer(line *parser.Line) error {
	return parseMaybeString(line)
}

func parseRun(line *parser.Line) error {
	return parseCmdEntrypointRun(line)
}

func parseShell(line *parser.Line) error {
	if !regJSONArray.MatchString(line.Raw) {
		return errors.Errorf("parse SHELL failed, line content is not JSON format, line: %s %s", line.Command, line.Raw)
	}

	fields, err := parseJSONArray(line.Raw)
	if err != nil {
		return err
	}
	addFieldsToLine(line, fields)

	// SHELL has at least one argument, e.g. SHELL ["executable", "parameters"]
	if len(line.Cells) < 1 {
		return errors.New("parse failed, SHELL requires at least one argument")
	}

	return nil
}

func parseOnBuild(line *parser.Line) error {
	flags := line.Raw
	fields := strings.Fields(flags)
	// ONBUILD has at least one argument, e.g. ONBUILD [INSTRUCTION]
	if len(fields) < 1 {
		return errors.New("ONBUILD command requires at least one argument")
	}
	cmd := strings.ToUpper(fields[0])
	if cmd == "ONBUILD" || cmd == "FROM" || cmd == "MAINTAINER" {
		return errors.Errorf("%q isn't allowed as an ONBUILD trigger", cmd)
	}
	cell := &parser.Cell{
		Value: cmd,
	}
	line.AddCell(cell)

	if _, ok := preHandlers[cmd]; !ok {
		return errors.Errorf("%q isn't support", cmd)
	}

	// rewrite Raw and Command for sub parse
	line.Raw = strings.Join(fields[1:], " ")
	line.Command = fields[0]
	err := preHandlers[cmd](line)

	// recover Raw and Command for this line
	line.Raw = flags
	line.Command = "ONBUILD"

	return err
}

func parseVolume(line *parser.Line) error {
	if err := parseAddCopyVolume(line, Volume); err != nil {
		return err
	}

	// VOLUME has at least one argument, e.g. VOLUME ["<path1>", "<path2>"...]
	if len(line.Cells) < 1 {
		return errors.New("parse failed, VOLUME requires at least one argument")
	}

	return nil
}

func parseStopSignal(line *parser.Line) error {
	return parseMaybeString(line)
}

func parseUser(line *parser.Line) error {
	return parseMaybeString(line)
}

func parseWorkDir(line *parser.Line) error {
	return parseMaybeString(line)
}

func extractFlags(line *parser.Line, cmd string) (string, error) {
	flagRegs := cmdFlagRegs[cmd]
	parts := strings.Fields(line.Raw)

	existFlags := make(map[string]bool, 0)
	var i int
	for ; i <= len(parts)-1; i++ {
		if !strings.HasPrefix(parts[i], "--") {
			break
		}
		kv := strings.SplitN(parts[i], "=", 2)
		if len(kv) < 2 {
			return "", errors.Errorf("%q should has specified value with '='", parts[i])
		}
		flagName := strings.TrimPrefix(kv[0], "--")
		if _, ok := flagRegs[flagName]; !ok {
			return "", errors.Errorf("unknown flag %q for command %q", flagName, cmd)
		}
		reg := flagRegs[flagName]
		if reg != nil && !reg.MatchString(kv[1]) {
			return "", errors.Errorf("invalid flag %s in line: %s %s", kv, line.Command, line.Raw)
		}
		if _, ok := existFlags[flagName]; ok {
			return "", errors.Errorf("duplicate flag %s in line: %s %s", flagName, line.Command, line.Raw)
		}
		if flagName == "retries" {
			if retries, err := strconv.Atoi(kv[1]); err != nil {
				return "", err
			} else if retries <= 0 {
				return "", errors.Errorf("healthcheck retries must be at least 1, here is %d", retries)
			}
		}
		existFlags[flagName] = true
		line.Flags[flagName] = kv[1]
	}

	lineWithoutCmdFlags := strings.Join(parts[i:], " ")

	return lineWithoutCmdFlags, nil
}

func addFieldsToLine(line *parser.Line, fields []string) {
	for _, field := range fields {
		cell := &parser.Cell{
			Value: field,
		}
		line.AddCell(cell)
	}
}

func parseLineWithWhiteSpace(line *parser.Line) {
	fields := strings.Fields(line.Raw)
	addFieldsToLine(line, fields)
}

func parseMaybeString(line *parser.Line) error {
	cmd := line.Command
	if line.Raw == "" {
		return errors.Errorf("%s command requires at least one argument", cmd)
	}
	// pre-process
	switch cmd {
	case Maintainer, WorkDir:
		cell := &parser.Cell{
			Value: line.Raw,
		}
		line.AddCell(cell)
	default:
		if cmdFlag := regCmdFlag.FindString(line.Raw); cmdFlag != "" {
			return errors.Errorf("invalid flag %s in line: %s %s", cmdFlag, line.Command, line.Raw)
		}
		parseLineWithWhiteSpace(line)
	}

	// cmd validation
	switch cmd {
	case From:
		// FROM has one or three arguments, e.g. FROM <image> [AS <name>]
		if len(line.Cells) != 1 && len(line.Cells) != 3 {
			return errors.New("FROM requires one argument, or three: FROM <source> [AS <name>]")
		}
	case StopSignal, User:
		// STOPSIGNAL and USER has only one arguments, e.g.
		// STOPSIGNAL signal
		// USER <user>[:<group>]
		if len(line.Cells) != 1 {
			return errors.Errorf("%s requires only one argument", cmd)
		}
	}

	return nil
}

func parseAddCopyVolume(line *parser.Line, cmd string) error {
	lineWithoutCmdFlags, err := extractFlags(line, cmd)
	if err != nil {
		return err
	}

	// check lineContent is not empty
	if lineWithoutCmdFlags == "" {
		return errors.Errorf("parse failed, line content is empty except command flags, line: %s %s", line.Command, line.Raw)
	}

	if regJSONArray.MatchString(lineWithoutCmdFlags) {
		if fields, err := parseJSONArray(lineWithoutCmdFlags); err == nil {
			line.Flags["attribute"] = "json"
			addFieldsToLine(line, fields)
			return nil
		} else if err == errJSONArrayIsNotString {
			return err
		}
	}

	addFieldsToLine(line, strings.Fields(lineWithoutCmdFlags))

	return nil
}

func parseCmdEntrypointRun(line *parser.Line) error {
	if line.Raw == "" {
		return nil
	}

	if cmdFlag := regCmdFlag.FindString(line.Raw); cmdFlag != "" {
		return errors.Errorf("invalid flag %s in line: %s %s", cmdFlag, line.Command, line.Raw)
	}

	var (
		fields []string
		err    error
	)
	if regJSONArray.MatchString(line.Raw) {
		if fields, err = parseJSONArray(line.Raw); err == nil {
			line.Flags["attribute"] = "json"
			addFieldsToLine(line, fields)
			return nil
		}

		if err == errJSONArrayIsNotString {
			return err
		}
	}

	fields = append(fields, line.Raw)
	addFieldsToLine(line, fields)

	return nil
}

func parseJSONArray(lineContent string) ([]string, error) {
	var (
		fields      []string
		JSONContent []interface{}
	)

	if err := json.NewDecoder(strings.NewReader(lineContent)).Decode(&JSONContent); err != nil {
		return nil, err
	}

	for _, JSONValue := range JSONContent {
		switch v := JSONValue.(type) {
		case string:
			if v != "" {
				fields = append(fields, v)
			}
		default:
			return nil, errJSONArrayIsNotString
		}
	}

	return fields, nil
}

// parse line content like <key>[=value], the value part is not necessary
func parseKeyOrKeyValue(line *parser.Line) error {
	fields := regKeyOrKeyValue.FindStringSubmatch(line.Raw)
	if fields == nil {
		return errors.Errorf("parse %d-%d %s with <key>[=value] format failed", line.Begin, line.End, line.Raw)
	}
	for i := 0; i <= len(fields)-1; i++ {
		fields[i] = strings.TrimSpace(fields[i])
	}
	value := strings.Join(fields[1:], "")
	cell := &parser.Cell{
		Value: value,
	}
	line.AddCell(cell)

	return nil
}

// parse line content like <key> <value> or <key>=<value> ...
// if is <key> <value> format, the <value> is must not be empty
// if is <key>=<value> format, is ok when the <value> is empty, some space
// between <key> and = or = and <value> is also right
func parseKeyValue(line *parser.Line) error {
	fieldsWithKV := regKeyValue.FindStringSubmatch(line.Raw)
	if fieldsWithKV != nil {
		fieldsWithKV[2] = strconv.Quote(fieldsWithKV[2])
		cell := &parser.Cell{
			Value: strings.Join(fieldsWithKV[1:], "="),
		}
		line.AddCell(cell)
		return nil
	}

	kvPairs := parseKeyEqualValuePairs(line.Raw)
	if len(kvPairs) == 0 {
		return errors.Errorf("parse %d-%d %s with key=value or key value format failed", line.Begin, line.End, line.Raw)
	}

	for _, kv := range kvPairs {
		parts := strings.SplitN(kv, "=", 2)
		if len(strings.Fields(parts[0])) > 1 {
			return errors.Errorf("syntax error: %s, is not a valid key=value format", kv)
		}
		cell := &parser.Cell{
			Value: kv,
		}
		line.AddCell(cell)
	}

	return nil
}

func parseKeyEqualValuePairs(str string) []string {
	kvPairs := make([]string, 0, 0)

	for i := 0; i <= len(str)-1; i++ {
		word := []byte{}

		// 1. scan str until the '=', and this time the word is like 'k='
		findEqualChar := false
		for ; i <= len(str)-1; i++ {
			word = append(word, str[i])
			if str[i] == '=' {
				findEqualChar = true
				word = bytes.TrimSpace(word[:len(word)-1])
				word = append(word, str[i])
				i++
				break
			}
		}
		if !findEqualChar {
			return nil
		}

		// 2. is the 'value' start with quote char
		isQuoteWord := false
		if i <= len(str)-1 && str[i] == '"' {
			word = append(word, str[i])
			isQuoteWord = true
			i++
		}

		// 3. scan for the value
		for ; i <= len(str)-1; i++ {
			if !isQuoteWord && unicode.IsSpace(rune(str[i])) {
				break
			}
			word = append(word, str[i])
			if isQuoteWord && str[i] == '"' && str[i-1] != '\\' {
				break
			}
		}
		kvPairs = append(kvPairs, string(word))
	}

	return kvPairs
}
