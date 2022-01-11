#!/bin/bash

# Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
# isula-build licensed under the Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#     http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v2 for more details.
# Author: Danni Xia
# Create: 2020-03-01
# Description: common functions for tests
# History: 2022-01-10 Weizheng Xing <xingweizheng@huawei.com> Refactor: only maintain the most common functions here

# exit_flag for a testcase, which will be added one when check or command goes something wrong
exit_flag=0

# show command brief and run
# $1 (command brief)
# $2 (concrete command)
function show_and_run_command() {
    function run_command() {
        if ! $command >/tmp/buildlog-client 2>&1; then
            echo "FAIL"
            echo "Failed when running command: $command"
            echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon"
            kill -15 "${pidofbuilder}"
            shell_print_callstack
            exit 1
        fi
    }
    if [ $# -eq 1 ]; then
        local -r command="$1"
        run_command
        return 0
    fi
    local -r brief="$1"
    local -r command="$2"
    printf "%-45s" "$brief:"
    run_command
    echo "PASS"
}

# run command when isula-builder running in systemd mode
# $1 (concrete command)
function systemd_run_command() {
    local -r command="$1"

    start_time=$(date '+%Y-%m-%d %H:%M:%S')
    if ! $command >/tmp/buildlog-client 2>&1; then
        {
            echo "Error from client:"
            cat /tmp/buildlog-client
            echo "Error from daemon:"
            journalctl -u isula-build --since "$start_time" --no-pager
            shell_print_callstack
        } >>/tmp/buildlog-failed

        ((exit_flag++))
    fi
}

# run command and check its result
# $1 (command)
# $2 (expected command return value)
function run_check_result() {
    local -r command="$1"
    local -r expected="$2"

    eval "$command" >/dev/null 2>&1
    result=$?
    debug "expected $expected, get $result"
    if [ "$result" != "$expected" ]; then
        testcase_path="${BASH_SOURCE[1]}"
        testcase="${testcase_path##/*/}"
        {
            echo "$testcase:${BASH_LINENO[0]}" "$command"
            echo expected "$expected", get "$result"
        } >>/tmp/buildlog-failed
        ((exit_flag++))
    fi
}

# check actual result and expected value
# $1 (result)
# $2 (expected)
function check_value() {
    local -r result="$1"
    local -r expected="$2"
    debug "expected $expected, get $result"

    if [ "$result" != "$expected" ]; then
        testcase_path="${BASH_SOURCE[1]}"
        testcase="${testcase_path##/*/}"
        {
            echo "TESTCASE: $testcase:${BASH_LINENO[0]}" "${FUNCNAME[0]}"
            echo expected "$expected", get "$result"
        } >>/tmp/buildlog-failed
        ((exit_flag++))
    fi
}

# print debug message
# $1 (debug message)
function debug() {
    local -r message="$1"

    if [ "$TEST_DEBUG" == "true" ]; then
        printf "(%s %s)  " "DEBUG:" "$message"
    fi
}

# print callstack of shell function
function shell_print_callstack() {
    echo "Shell Function Call Stack:"
    for ((index = 1; index < ${#FUNCNAME[@]}; index++)); do
        printf "\t%d - %s\n" "$((index - 1))" "${FUNCNAME[index]} (${BASH_SOURCE[index + 1]}:${BASH_LINENO[index]})"
    done
}
