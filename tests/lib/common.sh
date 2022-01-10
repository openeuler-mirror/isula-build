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

# cross process environment for killing isula-builder
declare -x pidofbuilder

# check if legacy builder exists
function pre_check() {
    if pgrep isula-builder >/dev/null 2>&1; then
        echo "isula-builder is already running, please stop it first"
        exit 1
    fi
}

# start isula-builder
function start_isula_builder() {
    nohup isula-builder >/tmp/buildlog-daemon 2>&1 &
    pidofbuilder=$!

    # check if isula-builder is started
    builder_started=0
    for _ in $(seq 1 30); do
        if ! grep -i "is listening on" /tmp/buildlog-daemon >/dev/null 2>&1; then
            sleep 0.1
            continue
        else
            builder_started=1
            break
        fi
    done
    if [ "${builder_started}" -eq 0 ]; then
        echo "isula-builder start failed, log dir /tmp/buildlog-daemon"
        exit 1
    fi
}

function cleanup() {
    isula-build ctr-img rm -p >/dev/null 2>&1
    kill -15 "${pidofbuilder}" >/dev/null 2>&1
    rm -f /tmp/buildlog-*
}

# test build image without output with default docker format
function test_build_without_output() {
    commands=(
        "isula-build ctr-img build --format docker --tag $1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )

    for command in "${commands[@]}"; do show_and_run_command "$command"; done
}

# test build image without output with oci format
function test_build_without_output_with_oci_format() {
    declare -a commands=(
        "isula-build ctr-img build --format oci --tag $1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
}

# test build image with docker-archive output
function test_build_with_docker_archive_output() {
    declare -a commands=(
        "isula-build ctr-img build --output=docker-archive:/tmp/$1.tar:$1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    rm -f /tmp/"$1".tar
}

# test build image with oci-archive output
function test_build_with_oci_archive_output() {
    declare -a commands=(
        "isula-build ctr-img build --output=oci-archive:/tmp/$1.tar:$1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    rm -f /tmp/"$1".tar
}

# test build image with docker-daemon output
function test_build_with_docker_daemon_output() {
    if ! systemctl status docker >/dev/null 2>&1; then
        return 0
    fi

    declare -a commands=(
        "isula-build ctr-img build --output=docker-daemon:isula/$1:latest $2"
        "isula-build ctr-img rm isula/$1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    docker rmi isula/"$1" >/dev/null 2>&1
}

# test build image with isulad output
function test_build_with_isulad_output() {
    if ! systemctl status isulad >/dev/null 2>&1; then
        return 0
    fi

    declare -a commands=(
        "isula-build ctr-img build --output=isulad:isula/$1:latest $2"
        "isula-build ctr-img rm isula/$1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    isula rmi isula/"$1" >/dev/null 2>&1
}

# test isula build base command
function test_isula_build_base_command() {
    declare -A commands=(
        ["Build docker format image"]="isula-build ctr-img build --tag $1-docker:latest --output=docker-archive:/tmp/$1-docker.tar:$1-docker:latest $2"
        ["Build oci format image"]="isula-build ctr-img build --tag $1-oci:latest --output=oci-archive:/tmp/$1-oci.tar:$1-oci:latest $2"
        ["List all images"]="isula-build ctr-img images"
        ["List docker format image"]="isula-build ctr-img images $1-docker:latest"
        ["List oci format image"]="isula-build ctr-img images $1-oci:latest"
        ["Save image with docker format"]="isula-build ctr-img save -f docker $1-docker:latest -o /tmp/$1-save-docker.tar"
        ["Save image with oci format"]="isula-build ctr-img save -f oci $1-oci:latest -o /tmp/$1-save-oci.tar"
        ["Load docker format images"]="isula-build ctr-img load -i /tmp/$1-docker.tar"
        ["Load oci format images"]="isula-build ctr-img load -i /tmp/$1-oci.tar"
        ["Save multipile images with docker format"]="isula-build ctr-img save -f docker $1-docker:latest $1-oci:latest -o /tmp/$1-all.tar"
        ["Remove images"]="isula-build ctr-img rm $1-docker:latest $1-oci:latest"
    )
    declare -a orders
    orders+=("Build docker format image")
    orders+=("Build oci format image")
    orders+=("List all images")
    orders+=("List docker format image")
    orders+=("List oci format image")
    orders+=("Save image with docker format")
    orders+=("Save image with oci format")
    orders+=("Load docker format images")
    orders+=("Load oci format images")
    orders+=("Save multipile images with docker format")
    orders+=("Remove images")
    for i in "${!orders[@]}"; do
        show_and_run_command "${orders[$i]}" "${commands[${orders[$i]}]}"
    done

    rm -f /tmp/*.tar
}

# print callstack of shell function
function shell_print_callstack() {
    echo "Shell Function Call Stack:"
    for ((index = 1; index < ${#FUNCNAME[@]}; index++)); do
        printf "\t%d - %s\n" "$((index - 1))" "${FUNCNAME[index]} (${BASH_SOURCE[index + 1]}:${BASH_LINENO[index]})"
    done
}

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
    printf "%-45s" "$brief"":"
    run_command
    echo "PASS"
}

exit_flag=0

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
    if [ "$result" != "$expected" ]; then
        debug "expected $expected, get $result"
        testcase_path="${BASH_SOURCE[1]}"
        testcase="${testcase_path##/*/}"
        echo "$testcase:${BASH_LINENO[0]}" "$command" >>/tmp/buildlog-failed
        echo expected "$expected", get "$result" >>/tmp/buildlog-failed
        ((exit_flag++))
    fi
}

# check actual result and expected value
# $1 (result)
# $2 (expected)
function check_value() {
    local -r result="$1"
    local -r expected="$2"

    if [ "$result" != "$expected" ]; then
        debug "expected $expected, get $result"
        testcase_path="${BASH_SOURCE[1]}"
        testcase="${testcase_path##/*/}"
        echo "TESTCASE: $testcase:${BASH_LINENO[0]}" "${FUNCNAME[0]}" >>/tmp/buildlog-failed
        echo expected "$expected", get "$result" >>/tmp/buildlog-failed
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

run_root="/var/run/integration-isula-build"
data_root="/var/lib/integration-isula-build"
config_file="/etc/isula-build/configuration.toml"

function pre_integration() {
    rm -rf /tmp/buildlog-failed

    cp $config_file "$config_file".integration
    sed -i "/run_root/d;/data_root/d" $config_file
    echo "run_root = \"${run_root}\"" >>$config_file
    echo "data_root = \"${data_root}\"" >>$config_file

    systemctl restart isula-build
}

function after_integration() {
    systemd_run_command "isula-build ctr-img rm -a"

    rm -f $config_file
    mv "$config_file".integration $config_file
    systemctl stop isula-build
    rm -rf $run_root $data_root
}
