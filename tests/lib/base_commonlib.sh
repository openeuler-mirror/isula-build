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
# Author: Weizheng Xing
# Create: 2022-01-10
# Description: common functions for isula-build base commands tests

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

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
    nohup isula-builder >"$TMPDIR"/buildlog-daemon 2>&1 &
    pidofbuilder=$!

    # check if isula-builder is started
    builder_started=0
    for _ in $(seq 1 30); do
        if ! grep -i "is listening on" "$TMPDIR"/buildlog-daemon >/dev/null 2>&1; then
            sleep 0.1
            continue
        else
            builder_started=1
            break
        fi
    done
    if [ "${builder_started}" -eq 0 ]; then
        echo "isula-builder start failed, log dir $TMPDIR/buildlog-daemon"
        cat "$TMPDIR"/buildlog-daemon
        exit 1
    fi
}

function cleanup() {
    isula-build ctr-img rm -p >/dev/null 2>&1
    kill -15 "${pidofbuilder}" >/dev/null 2>&1
    rm -rf "$TMPDIR"
}

# isula-build builds with different output
# $1 (image name)
# $2 (build context dir)
function test_isula_build_output() {
    local -r image_name="$1"
    local -r context_dir="$2"

    functions=(
        test_build_without_output_with_docker_format
        test_build_without_output_with_oci_format
        test_build_with_docker_archive_output
        test_build_with_oci_archive_output
        test_build_with_docker_daemon_output
        test_build_with_isulad_output
    )

    for function in "${functions[@]}"; do $function "$image_name" "$context_dir"; done
}

# test build image without output with default docker format
function test_build_without_output_with_docker_format() {
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
        "isula-build ctr-img build --output=docker-archive:$TMPDIR/$1.tar:$1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    rm -f "$TMPDIR"/"$1".tar
}

# test build image with oci-archive output
function test_build_with_oci_archive_output() {
    declare -a commands=(
        "isula-build ctr-img build --output=oci-archive:$TMPDIR/$1.tar:$1:latest $2"
        "isula-build ctr-img rm $1:latest"
    )
    for command in "${commands[@]}"; do show_and_run_command "$command"; done
    rm -f "$TMPDIR"/"$1".tar
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
        ["Build docker format image"]="isula-build ctr-img build --tag $1-docker:latest --output=docker-archive:$TMPDIR/$1-docker.tar:$1-docker:latest $2"
        ["Build oci format image"]="isula-build ctr-img build --tag $1-oci:latest --output=oci-archive:$TMPDIR/$1-oci.tar:$1-oci:latest $2"
        ["List all images"]="isula-build ctr-img images"
        ["List docker format image"]="isula-build ctr-img images $1-docker:latest"
        ["List oci format image"]="isula-build ctr-img images $1-oci:latest"
        ["Save image with docker format"]="isula-build ctr-img save -f docker $1-docker:latest -o $TMPDIR/$1-save-docker.tar"
        ["Save image with oci format"]="isula-build ctr-img save -f oci $1-oci:latest -o $TMPDIR/$1-save-oci.tar"
        ["Load docker format images"]="isula-build ctr-img load -i $TMPDIR/$1-docker.tar"
        ["Load oci format images"]="isula-build ctr-img load -i $TMPDIR/$1-oci.tar"
        ["Save multipile images with docker format"]="isula-build ctr-img save -f docker $1-docker:latest $1-oci:latest -o $TMPDIR/$1-all.tar"
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

    rm -f "$TMPDIR"/*.tar
}
