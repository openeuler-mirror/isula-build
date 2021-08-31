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
// Description: logger related functions

// Package logger is used to print log
package logger

import (
	"fmt"
	"sync"
	"time"
)

const maxContentChanSize = 100

// RunTimer stores time cost for commands
type RunTimer struct {
	lock     sync.Mutex
	commands []string
	cmdMap   map[string]time.Duration
}

// Timer stores each command's name and started time
type Timer struct {
	command   string
	startTime time.Time
}

// Logger logs message of which we want to print to the front end
type Logger struct {
	rt          *RunTimer
	content     chan string
	currentStep int
}

// NewRunTimer return an instance of RunTimer
func NewRunTimer() *RunTimer {
	return &RunTimer{
		commands: make([]string, 0),
		cmdMap:   make(map[string]time.Duration),
	}
}

// StartTimer starts record time
func (l *Logger) StartTimer(str string) *Timer {
	return &Timer{
		startTime: time.Now(),
		command:   str,
	}
}

// StopTimer stops record time for the command
func (l *Logger) StopTimer(t *Timer) {
	stop := time.Now()
	l.rt.lock.Lock()
	defer l.rt.lock.Unlock()
	l.rt.commands = append(l.rt.commands, fmt.Sprintf("%s: %s\n", t.command, stop.Sub(t.startTime).String()))
	if _, ok := l.rt.cmdMap[t.command]; !ok {
		l.rt.cmdMap[t.command] = 0
	}
	l.rt.cmdMap[t.command] += stop.Sub(t.startTime)
}

// GetCmdTime return one command consume time in the map
func (l *Logger) GetCmdTime(t *Timer) string {
	l.rt.lock.Lock()
	defer l.rt.lock.Unlock()
	if v, ok := l.rt.cmdMap[t.command]; ok {
		return fmt.Sprintf("%s: %s", t.command, v.String())
	}
	return ""
}

// Summary return time consumed during building
func (l *Logger) Summary() string {
	var summary string
	l.rt.lock.Lock()
	for _, v := range l.rt.commands {
		summary += v
	}
	l.rt.lock.Unlock()
	return summary
}

// Write is used to implement io.Writer
func (l *Logger) Write(p []byte) (int, error) {
	l.content <- string(p)
	return len(p), nil
}

// StepPrint can be only used to print step info in each command line of the dockerfile
func (l *Logger) StepPrint(str string) {
	l.currentStep++
	content := fmt.Sprintf("STEP %2d: %s\n", l.currentStep, str)
	l.content <- content
}

// Print transport message to the front in the client end
func (l *Logger) Print(format string, a ...interface{}) {
	l.content <- fmt.Sprintf(format, a...)
}

// CloseContent close channel connected with frontend
func (l *Logger) CloseContent() {
	close(l.content)
}

// GetContent return content stored in channel
func (l *Logger) GetContent() <-chan string {
	return l.content
}

// GetStep return current step during building
func (l *Logger) GetStep() int {
	return l.currentStep
}

// NewCliLogger create an instance of Logger
func NewCliLogger(len int) *Logger {
	if len > maxContentChanSize {
		len = maxContentChanSize
	}

	return &Logger{
		rt:          NewRunTimer(),
		content:     make(chan string, len),
		currentStep: 0,
	}
}
