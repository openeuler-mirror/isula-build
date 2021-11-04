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
# Author: Xiang Li
# Create: 2021-11-01
# Description: cover test for save separated image

test_name=${BASH_SOURCE[0]}
workspace=/tmp/${test_name}.$(date +%s)
mkdir -p "${workspace}"
dockerfile=${workspace}/Dockerfile
top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "${top_dir}"/tests/lib/separator.sh

function pre_run() {
    base_image_name="hub.oepkgs.net/library/busybox:latest"
    lib_image_name="lib:latest"
    app1_image_name="app1:latest"
    app2_image_name="app2:latest"
    base_image_short_name="b:latest"
    lib_image_short_name="l:latest"
    app1_image_short_name="a:latest"
    app2_image_short_name="c:latest"

    lib_layer_number=5
    app1_layer_number=4
    app2_layer_number=3
    touch_dockerfile "${base_image_name}" "${lib_image_name}" "${lib_layer_number}" "${dockerfile}"
    build_image "${lib_image_name}" "${workspace}"
    touch_dockerfile "${lib_image_name}" "${app1_image_name}" "${app1_layer_number}" "${dockerfile}"
    build_image "${app1_image_name}" "${workspace}"
    touch_dockerfile "${lib_image_name}" "${app2_image_name}" "${app2_layer_number}" "${dockerfile}"
    build_image "${app2_image_name}" "${workspace}"
}

function test_run1() {
    isula-build ctr-img save -b "${base_image_name}" -l "${lib_image_name}" -d "${workspace}"/Images "${app1_image_name}" "${app2_image_name}"
    check_result_equal $? 0
    rm -rf "${workspace}"
}

# use short image name
function test_run2() {
    isula-build ctr-img tag "${base_image_name}" "${base_image_short_name}"
    isula-build ctr-img tag "${lib_image_name}" "${lib_image_short_name}"
    isula-build ctr-img tag "${app1_image_name}" "${app1_image_short_name}"
    isula-build ctr-img tag "${app2_image_name}" "${app2_image_short_name}"
    isula-build ctr-img save -b "${base_image_short_name}" -l "${lib_image_short_name}" -d "${workspace}"/Images "${app1_image_short_name}" "${app2_image_short_name}"
    check_result_equal $? 0
    rm -rf "${workspace}"
}

function cleanup() {
    rm -rf "${workspace}"
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}" "${base_image_short_name}" "${lib_image_short_name}" "${app1_image_short_name}" "${app2_image_short_name}"
}

pre_run
test_run1
test_run2
cleanup
# shellcheck disable=SC2154
exit "${exit_flag}"
