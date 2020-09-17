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

# test build image without output
function test_build_without_output() {
    if ! isula-build ctr-img build --tag "$1":latest "$2" > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build without output)"
        kill -15 "${pidofbuilder}"
        exit 1
    fi

    if ! isula-build ctr-img rm "$1":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build without output)"
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

# test build image with docker-daemon output
function test_build_with_docker_daemon_output() {
    systemctl status docker > /dev/null 2>&1
    if [ $? -ne 0 ]; then
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
    systemctl status isulad > /dev/null 2>&1
    if [ $? -ne 0 ]; then
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
