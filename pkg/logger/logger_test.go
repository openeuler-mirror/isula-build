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
// Create: 2020-03-20
// Description: logger related functions tests

package logger

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"gotest.tools/assert"

	constant "isula.org/isula-build"
)

func TestLoggerPrint(t *testing.T) {
	type args struct {
		format string
		a      []interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Print with one sentence",
			args: args{
				format: "This is a sentence which will print to the cli frontend",
				a:      nil,
			},
		},
		{
			name: "Print with long sentences",
			args: args{
				format: "This is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\nThis is a sentence which will print to the cli frontend\n",
				a:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewCliLogger(constant.CliLogBufferLen)
			l.Print(tt.args.format)
			content, ok := <-l.GetContent()
			assert.Equal(t, ok, true)
			assert.Equal(t, tt.args.format, content)
		})
	}
}

func TestLoggerStepPrint(t *testing.T) {
	type args struct {
		format string
	}
	tests := []struct {
		name      string
		cliLogger *Logger
		args      args
	}{
		{
			name: "StepPrint without step",
			args: args{
				format: "FROM alpine AS cho",
			},
			cliLogger: NewCliLogger(constant.CliLogBufferLen),
		},
		{
			name: "StepPrint with step 1",
			args: args{
				format: "CMD ls",
			},
			cliLogger: NewCliLogger(constant.CliLogBufferLen),
		},
		{
			name: "StepPrint with no content",
			args: args{
				format: "",
			},
			cliLogger: NewCliLogger(constant.CliLogBufferLen),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.cliLogger
			l.StepPrint(tt.args.format)
			content, ok := <-l.GetContent()
			assert.Equal(t, ok, true)
			assert.Equal(t, fmt.Sprintf("STEP %2d: %s", l.GetStep(), tt.args.format+"\n"), content)
		})
	}
}

func TestLoggerStartTimer(t *testing.T) {
	type fields struct {
		rt          *RunTimer
		content     chan string
		currentStep int
	}

	type args struct {
		str string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expect *regexp.Regexp
	}{
		{
			name: "One function called",
			fields: fields{
				rt:          NewRunTimer(),
				content:     nil,
				currentStep: 0,
			},
			args:   args{str: "Sleep 1s"},
			expect: regexp.MustCompile(`Sleep 1s: 1..*s\n`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{
				rt:          tt.fields.rt,
				content:     tt.fields.content,
				currentStep: tt.fields.currentStep,
			}
			timer := l.StartTimer(tt.args.str)
			time.Sleep(time.Second)
			l.StopTimer(timer)
			result := l.Summary()
			matched := tt.expect.MatchString(result)
			assert.Equal(t, matched, true)
		})
	}
}

func TestLoggerStartTimerForMultipleFunc(t *testing.T) {
	rt := NewRunTimer()
	l := &Logger{
		rt:          rt,
		content:     nil,
		currentStep: 0,
	}
	timer1 := l.StartTimer("Sleep 1s")
	time.Sleep(time.Second)
	l.StopTimer(timer1)
	timer2 := l.StartTimer("Sleep 2s")
	time.Sleep(time.Second)
	time.Sleep(time.Second)
	l.StopTimer(timer2)
	result := l.Summary()
	expect := regexp.MustCompile(`Sleep 1s: 1..*s\nSleep 2s: 2..*s\n`)
	matched := expect.MatchString(result)
	assert.Equal(t, matched, true)
}

func TestLoggerGetCmdTime(t *testing.T) {
	rt := NewRunTimer()
	l := &Logger{
		rt:          rt,
		content:     nil,
		currentStep: 0,
	}
	emptyTimer := &Timer{
		startTime: time.Now(),
		command:   "cmd not exist",
	}
	timer := l.StartTimer("Sleep 1s")
	time.Sleep(time.Second)
	l.StopTimer(timer)
	result := l.GetCmdTime(timer)
	expect := regexp.MustCompile(`Sleep 1s: 1..*s`)
	matched := expect.MatchString(result)
	assert.Equal(t, matched, true)
	emtyeResult := l.GetCmdTime(emptyTimer)
	assert.Equal(t, emtyeResult, "")
}
