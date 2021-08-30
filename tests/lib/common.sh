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
    if pgrep isula-builder > /dev/null 2>&1; then
        echo "isula-builder is already running, please stop it first"
        exit 1
    fi
}

# start isula-builder
function start_isula_builder() {
    nohup isula-builder > /tmp/buildlog-daemon 2>&1 &
    pidofbuilder=$!

    # check if isula-builder is started
    builder_started=0
    for _ in $(seq 1 30); do
        if ! grep -i "is listening on" /tmp/buildlog-daemon > /dev/null 2>&1; then
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
    isula-build ctr-img rm -p > /dev/null 2>&1
    kill -15 "${pidofbuilder}" > /dev/null 2>&1
    rm -f /tmp/buildlog-*
}

# test build image without output with default docker format
function test_build_without_output() {
    if ! isula-build ctr-img build --format docker --tag "$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build  with docker format without output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi

    if ! isula-build ctr-img rm "$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build  with docker format without output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test build image without output with oci format
function test_build_without_output_with_oci_format() {
    if ! isula-build ctr-img build --format oci --tag "$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with oci format without output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi

    if ! isula-build ctr-img rm "$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with oci format without output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test build image with docker-archive output
function test_build_with_docker_archive_output() {
    if ! isula-build ctr-img build --output=docker-archive:/tmp/"$1".tar:"$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-archive output)"
        kill -15 "${pidofbuilder}"
        exit 1
    else
        rm -f /tmp/"$1".tar
    fi

    if ! isula-build ctr-img rm "$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-archive output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test build image with oci-archive output
function test_build_with_oci_archive_output() {
    if ! isula-build ctr-img build --output=oci-archive:/tmp/"$1".tar:"$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with oci-archive output)"
        kill -15 "${pidofbuilder}"
        exit 1
    else
        rm -f /tmp/"$1".tar
    fi

    if ! isula-build ctr-img rm "$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with oci-archive output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test build image with docker-daemon output
function test_build_with_docker_daemon_output() {
    if ! systemctl status docker > /dev/null 2>&1; then
        return 0
    fi

    if ! isula-build ctr-img build --output=docker-daemon:isula/"$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-daemon output)"
        kill -15 "${pidofbuilder}"
        exit 1
    else
        docker rmi isula/"$1" > /dev/null 2>&1
    fi

    if ! isula-build ctr-img rm isula/"$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-daemon output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test build image with isulad output
function test_build_with_isulad_output() {  
    if ! systemctl status isulad > /dev/null 2>&1; then
        return 0
    fi

    if ! isula-build ctr-img build --output=isulad:isula/"$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with isulad output)"
        kill -15 "${pidofbuilder}"
        exit 1
    else
        isula rmi isula/"$1" > /dev/null 2>&1
    fi

    if ! isula-build ctr-img rm isula/"$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with isulad output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
}

# test isula build base command
function test_isula_build_base_command() {
    show_and_run_command "Build docker format image:" \
    " isula-build ctr-img build --tag $1-docker:latest --output=docker-archive:/tmp/$1-docker.tar:$1-docker:latest $2"

    show_and_run_command "Build oci format image:" \
    "isula-build ctr-img build --tag $1-oci:latest --output=oci-archive:/tmp/$1-oci.tar:$1-oci:latest $2"

    show_and_run_command "List all images:" \
    "isula-build ctr-img images"
    
    show_and_run_command "List docker format image:" \
    "isula-build ctr-img images $1-docker:latest"

    show_and_run_command "List oci format image:" \
    "isula-build ctr-img images $1-oci:latest"

    rm -f /tmp/"$1"-docker.tar /tmp/"$1"-oci.tar

    show_and_run_command "Save image with docker format:" \
    "isula-build ctr-img save -f docker $1-docker:latest -o /tmp/$1-docker.tar"

    show_and_run_command "Save image with oci format:" \
    "isula-build ctr-img save -f oci $1-oci:latest -o /tmp/$1-oci.tar"

    show_and_run_command "Load docker format images:" \
    "isula-build ctr-img load -i /tmp/$1-docker.tar"

    show_and_run_command "Load oci format images:" \
    "isula-build ctr-img load -i /tmp/$1-oci.tar"

    show_and_run_command "Save multipile images with docker format:" \
    "isula-build ctr-img save -f docker $1-docker:latest $1-oci:latest -o /tmp/$1-all.tar"

    rm -f /tmp/"$1"-docker.tar /tmp/"$1"-oci.tar /tmp/"$1"-all.tar

    show_and_run_command "Remove images:" \
    "isula-build ctr-img rm $1-docker:latest $1-oci:latest"
}

function show_and_run_command() {
    printf "%-45s" "$1"
    if ! $2 > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon, failed when running command: $2"
        kill -15 "${pidofbuilder}"
        exit 1
    fi
    echo "PASS"
}

function run_with_debug() {
    if [ "${DEBUG:-0}" -eq 1 ]; then
        $1
    else
        $1 > /dev/null 2>&1
    fi
}