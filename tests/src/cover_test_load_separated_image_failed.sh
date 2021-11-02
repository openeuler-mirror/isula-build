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

# empty -d flag and missing -b
function test_run1() {
    isula-build ctr-img load -l "${tarball_dir}"/base1.tar.gz -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# empty -d flag and missing -l
function test_run2() {
    isula-build ctr-img load -b "${tarball_dir}"/base1.tar.gz -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# empty -d, -b, -l flag
function test_run3() {
    isula-build ctr-img load -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# use lib as base image tarball
function test_run4() {
    isula-build ctr-img load -d "${tarball_dir}" -b "${tarball_dir}"/lib1.tar.gz -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# missing app tarball
function test_run5() {
    mv "${tarball_dir}"/app1.tar.gz "${workspace}"
    isula-build ctr-img load -d "${tarball_dir}" -l "${tarball_dir}"/lib1.tar.gz -i "${app1_image_name}"
    check_result_not_equal $? 0
    mv "${workspace}"/app1.tar.gz "${tarball_dir}"
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# lib tarball not exist
function test_run6() {
    isula-build ctr-img load -d "${tarball_dir}" -l not_exist_lib.tar -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# base tarball not exist
function test_run7() {
    isula-build ctr-img load -d "${tarball_dir}" -b not_exist_base.tar -i "${app1_image_name}"
    check_result_not_equal $? 0
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# invalid base tarball
function test_run8() {
    invalid_tarball=${workspace}/base1.tar
    echo "invalid base tarball" >> "${invalid_tarball}"
    isula-build ctr-img load -d "${tarball_dir}" -b "${invalid_tarball}" -i "${app1_image_name}"
    check_result_not_equal $? 0
    rm -rf "${invalid_tarball}"
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# invalid lib tarball
function test_run9() {
    invalid_tarball=${workspace}/lib1.tar
    echo "invalid lib tarball" >> "${invalid_tarball}"
    isula-build ctr-img load -d "${tarball_dir}" -l "${invalid_tarball}" -i "${app1_image_name}"
    check_result_not_equal $? 0
    rm -rf "${invalid_tarball}"
    isula-build ctr-img rm "${lib_image_name}" "${app1_image_name}" "${app2_image_name}"
}

# manifest file corruption
function test_run10() {
    cp "${tarball_dir}"/manifest "${tarball_dir}"/manifest.bk
    sed -i "1d" "${tarball_dir}"/manifest
    isula-build ctr-img load -d "${tarball_dir}" -d "${tarball_dir}" -i "${app1_image_name}"
    check_result_not_equal $? 0
    mv "${tarball_dir}"/manifest.bk "${tarball_dir}"/manifest
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
test_run6
test_run7
test_run8
test_run9
test_run10
cleanup
# shellcheck disable=SC2154
exit "${exit_flag}"
