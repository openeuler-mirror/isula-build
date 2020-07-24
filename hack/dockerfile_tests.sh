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
# Description: shell script for dockerfile tests

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

# test build image without output
function test_build_without_output() {

    if ! isula-build ctr-img build > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build without output)"
        kill -9 "${pidofbuilder}"
        exit 1
    fi

}

# test build image with docker-archive output
function test_build_with_docker_archive_output() {
    if ! isula-build ctr-img build --output=docker-archive:/tmp/"${image_name}".tar > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-archive output)"
        kill -9 "${pidofbuilder}"
        exit 1
    else
        rm -f /tmp/"${dockerfiledir}".tar
    fi
}

function test_build_with_docker_daemon_output() {
    # test build image with docker-daemon output
    if ! isula-build ctr-img build --output=docker-daemon:isula/"${image_name}":latest > /tmp/buildlog-client 2>&1; then
        echo "FAIL"
        echo "LOG DIR:/tmp/buildlog-client and /tmp/buildlog-daemon (build with docker-daemon output)"
        kill -9 "${pidofbuilder}"
        exit 1
    else
        docker rmi isula/"${image_name}" > /dev/null 2>&1
    fi
}

# start build images tests
function tests() {
    while IFS= read -r dockerfiledir; do
        if ! find "${dockerfiledir}" -maxdepth 1 -iname "Dockerfile" | grep . > /dev/null 2>&1; then
            continue
        fi
        pushd "${dockerfiledir}" > /dev/null 2>&1 || exit 1
        image_name=$(basename "${dockerfiledir}")

        echo -e "test Dockerfile in ${dockerfiledir}:\c"
        test_build_without_output
        test_build_with_docker_archive_output
        test_build_with_docker_daemon_output
        echo "PASS"

        popd > /dev/null 2>&1 || exit 1
    done < <(find ./tests -maxdepth 1 -type d -print)
}

function cleanup() {
    isula-build ctr-img rm -p > /dev/null 2>&1
    kill -9 "${pidofbuilder}" > /dev/null 2>&1
    rm -f /tmp/buildlog-*
}


function main() {
    pre_check
    start_isula_builder
    tests
    cleanup
}

main
echo "SUCCESS: all tests success"
