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
# Create: 2021-08-24
# Description: check if saving single image with multiple tags has been corrected

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

image_name=add-chown-basic
context_dir="$top_dir"/tests/data/add-chown-basic

function clean()
{
    isula-build ctr-img rm -p > /dev/null 2>&1
    systemctl stop isula-build
    rm -rf "$temp_tar"
}

function pre_test()
{
    temp_tar=$(mktemp -u --suffix=.tar)
    systemctl restart isula-build
}

function do_test()
{
    # get image id
    if ! image_id1=$(isula-build ctr-img build -t $image_name:latest "$context_dir"|grep "Build success with image id: "|cut -d ":" -f 2); then
        echo "FAIL"
    fi
    if ! image_id2=$(isula-build ctr-img build -t $image_name:latest2 "$context_dir"|grep "Build success with image id: "|cut -d ":" -f 2); then
        echo "FAIL"
    fi

    ! run_with_debug "isula-build ctr-img tag $image_name:latest $image_name:latest-child"

    # save with id + name
    ! run_with_debug "isula-build ctr-img save -f docker $image_id1 $image_name:latest-child -o $temp_tar"
    rm -rf "$temp_tar"

    # save with name + id
    ! run_with_debug "isula-build ctr-img save -f docker $image_name:latest-child $image_id1 -o $temp_tar"
    rm -rf "$temp_tar"

    # save with name + name
    ! run_with_debug "isula-build ctr-img save -f docker $image_name:latest $image_name:latest-child -o $temp_tar"
    rm -rf "$temp_tar"

    # save with different images id1 + id2
    ! run_with_debug "isula-build ctr-img save -f docker $image_id1 $image_id2 -o $temp_tar"
    rm -rf "$temp_tar"

    # save with different images "without latest tag" + id2
    ! run_with_debug "isula-build ctr-img save -f docker $image_name $image_id2 -o $temp_tar"
    rm -rf "$temp_tar"

    # save with id1 + id2 + name
    ! run_with_debug "isula-build ctr-img save -f docker $image_id1 $image_id2 $image_name:latest2 -o $temp_tar"
    rm -rf "$temp_tar"

    ! run_with_debug "isula-build ctr-img rm $image_name:latest $image_name:latest-child"
    ! run_with_debug "isula-build ctr-img rm $image_name:latest2"

    echo "PASS" 
}

pre_test
do_test
clean
