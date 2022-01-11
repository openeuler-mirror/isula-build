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
# Description: test delete and save image well behaved when the [image]:tag of one image,
# image is prefix of other image id and tag is latest, such as [othershortid]:latest

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

image_name=build-from-scratch
context_dir="$top_dir"/tests/data/build-from-scratch

function pre_test() {
    temp_tar_short=$(mktemp -u --suffix=.tar)
    temp_tar_double_short=$(mktemp -u --suffix=.tar)
}

function clean() {
    systemd_run_command "isula-build ctr-img rm $image_name:latest2"
    rm -f "$temp_tar_short"
    rm -f "$temp_tar_double_short"
}

function do_test() {
    systemd_run_command "isula-build ctr-img build -t $image_name:latest1 $context_dir"
    systemd_run_command "isula-build ctr-img build -t $image_name:latest2 $context_dir"
    image_id2=$(grep <"$TMPDIR"/buildlog-client "Build success with image id: " | cut -d ":" -f 2)
    short_id2=${image_id2:0:12}
    double_short_id2=${short_id2:0:6}

    # get material
    declare -a commands=(
        "isula-build ctr-img tag $image_name:latest1 $short_id2"
        "isula-build ctr-img save -f docker $short_id2 -o $temp_tar_short"
        "isula-build ctr-img save -f docker $double_short_id2 -o $temp_tar_double_short"
        "isula-build ctr-img rm $short_id2"
        "isula-build ctr-img images $image_name:latest2"
    )
    for command in "${commands[@]}"; do systemd_run_command "$command"; done

    # analyse it
    declare -a commands=(
        "cat $TMPDIR/buildlog-client |grep $short_id2"
        "tar -xvf $temp_tar_short -C $TMPDIR manifest.json"
        "cat $TMPDIR/manifest.json | grep $short_id2:latest"
        "tar -xvf $temp_tar_double_short -C $TMPDIR manifest.json"
        "cat $TMPDIR/manifest.json | grep $image_id2"
    )
    for command in "${commands[@]}"; do run_check_result "$command" 0; done
}

pre_test
do_test
clean
exit "$exit_flag"
