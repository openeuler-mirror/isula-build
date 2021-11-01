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
# Description: cover test for load separated image

test_name=${BASH_SOURCE[0]}
workspace=/tmp/${test_name}.$(date +%s)
mkdir -p "${workspace}"
dockerfile=${workspace}/Dockerfile
tarball_dir=${workspace}/Images
rename_json=${workspace}/rename.json
top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "${top_dir}"/tests/lib/separator.sh

function pre_run() {
    base_image_name="hub.oepkgs.net/library/busybox:latest"
    lib_image_name="lib:latest"
    app1_image_name="app1:latest"
    app2_image_name="app2:latest"
    lib_layer_number=5
    app1_layer_number=4
    app2_layer_number=3
    touch_dockerfile "${base_image_name}" "${lib_image_name}" "${lib_layer_number}" "${dockerfile}"
    build_image "${lib_image_name}" "${workspace}"
    touch_dockerfile "${lib_image_name}" "${app1_image_name}" "${app1_layer_number}" "${dockerfile}"
    build_image "${app1_image_name}" "${workspace}"
    touch_dockerfile "${lib_image_name}" "${app2_image_name}" "${app2_layer_number}" "${dockerfile}"
    build_image "${app2_image_name}" "${workspace}"
    touch_rename_json "${rename_json}"
    isula-build ctr-img save -b "${base_image_name}" -l "${lib_image_name}" -d "${tarball_dir}" "${app1_image_name}" "${app2_image_name}" -r "${rename_json}"
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function test_run1() {
    isula-build ctr-img load -d "${tarball_dir}" -b "${tarball_dir}"/base1.tar.gz -l "${tarball_dir}"/lib1.tar.gz -i "${app1_image_name}"
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function test_run2() {
    isula-build ctr-img load -d "${tarball_dir}" -b "${tarball_dir}"/base1.tar.gz -i "${app1_image_name}"
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function test_run3() {
    isula-build ctr-img load -d "${tarball_dir}" -l "${tarball_dir}"/lib1.tar.gz -i "${app1_image_name}"
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function test_run4() {
    isula-build ctr-img load -d "${tarball_dir}" -i "${app1_image_name}"
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function test_run5() {
    isula-build ctr-img load -d "${tarball_dir}" -i "${app1_image_name}" --no-check
    check_result_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

function cleanup() {
    rm -rf "${workspace}"
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

pre_run
test_run1
test_run2
test_run3
test_run4
test_run5
cleanup
# shellcheck disable=SC2154
exit "${exit_flag}"
