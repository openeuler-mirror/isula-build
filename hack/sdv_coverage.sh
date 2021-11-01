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
# Create: 2020-03-01
# Description: shell script for coverage
# Note: use this file by typing make test-sdv-cover or make test-cover
#       Do not run this script directly

project_root=${PWD}
vendor_name="isula.org"
project_name="isula-build"
main_relative_path="cmd/daemon"
exclude_pattern="gopkgs|api/services"
go_test_mod_method="-mod=vendor"
go_test_count_method="-count=1"
go_test_cover_method="-covermode=set"
main_pkg="${vendor_name}/${project_name}/${main_relative_path}"
main_test_file=${project_root}/${main_relative_path}/main_test.go
main_file=${project_root}/${main_relative_path}/main.go
coverage_file=${project_root}/cover_sdv_test_all.out
coverage_html=${project_root}/cover_sdv_test_all.html
coverage_log=${project_root}/cover_sdv_test_all.log
main_test_binary_file=${project_root}/main.test

function precheck() {
    if pgrep isula-builder > /dev/null 2>&1; then
        echo "isula-builder is already running, please stop it first"
        exit 1
    fi
}

function modify_main_test() {
    # first backup file
    cp "${main_file}" "${main_file}".bk
    cp "${main_test_file}" "${main_test_file}".bk
    # delete Args field for main.go
    local comment_pattern="Args:  util.NoArgs"
    sed -i "/$comment_pattern/s/^#*/\/\/ /" "${main_file}"
    # add new line for main_test.go
    code_snippet="func TestMain(t *testing.T) { main() }"
    echo "$code_snippet" >> "${main_test_file}"
}

function recover_main_test() {
    mv "${main_file}".bk "${main_file}"
    mv "${main_test_file}".bk "${main_test_file}"
}

function build_main_test_binary() {
    pkgs=$(go list ${go_test_mod_method} "${project_root}"/... | grep -Ev ${exclude_pattern} | tr "\r\n" ",")
    go test -coverpkg="${pkgs}" ${main_pkg} ${go_test_mod_method} ${go_test_cover_method} ${go_test_count_method} -c -o="${main_test_binary_file}"
}

function run_main_test_binary() {
    ${main_test_binary_file} -test.coverprofile="${coverage_file}" > "${coverage_log}" 2>&1 &
    main_test_pid=$!
    for _ in $(seq 1 10); do
        if isula-build info > /dev/null 2>&1; then
            break
        else
            sleep 1
        fi
    done
}

function run_coverage_test() {
    # do cover tests
    echo "sdv coverage test"
    # cover_test_xxx
    # cover_test_xxx
    # cover_test_xxx
    # cover_test_xxx
}

function finish_coverage_test() {
    kill -15 $main_test_pid
}

function generate_coverage() {
    go tool cover -html="${coverage_file}" -o="${coverage_html}"
}

function cleanup() {
    rm "$main_test_binary_file"
}

precheck
modify_main_test
build_main_test_binary
recover_main_test
run_main_test_binary
run_coverage_test
finish_coverage_test
generate_coverage
cleanup
