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
# History: 2022-01-10 Weizheng Xing <xingweizheng@huawei.com> Refactor: use systemd_run_command common function

top_dir=$(git rev-parse --show-toplevel)
source "$top_dir"/tests/lib/common.sh

image_name=build-from-scratch
context_dir="$top_dir"/tests/data/build-from-scratch

function pre_test() {
    temp_tar=$(mktemp -u --suffix=.tar)
}

function do_test() {
    # get image id
    systemd_run_command "isula-build ctr-img build -t $image_name:latest $context_dir"
    image_id1=$(grep </tmp/buildlog-client "Build success with image id: " | cut -d ":" -f 2)

    systemd_run_command "isula-build ctr-img build -t $image_name:latest2 $context_dir"
    image_id2=$(grep </tmp/buildlog-client "Build success with image id: " | cut -d ":" -f 2)

    declare -a commands=(
        "isula-build ctr-img tag $image_name:latest $image_name:latest-child"
        # save with id + name
        "isula-build ctr-img save -f docker $image_id1 $image_name:latest-child -o $temp_tar"
        # save with name + id
        "isula-build ctr-img save -f docker $image_name:latest-child $image_id1 -o $temp_tar"
        # save with name + name
        "isula-build ctr-img save -f docker $image_name:latest $image_name:latest-child -o $temp_tar"
        # save with different images id1 + id2
        "isula-build ctr-img save -f docker $image_id1 $image_id2 -o $temp_tar"
        # save with different images "without latest tag" + id2
        "isula-build ctr-img save -f docker $image_name $image_id2 -o $temp_tar"
        # save with id1 + id2 + name
        "isula-build ctr-img save -f docker $image_id1 $image_id2 $image_name:latest2 -o $temp_tar"
        "isula-build ctr-img rm $image_name:latest $image_name:latest-child"
        "isula-build ctr-img rm $image_name:latest2"
    )
    for command in "${commands[@]}"; do
        systemd_run_command "$command"
        rm -rf "$temp_tar"
    done
}

pre_test
do_test
exit "$exit_flag"
