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
# Create: 2020-08-29
# Description: fuzz builder

top_dir=$(git rev-parse --show-toplevel)
test_name="fuzz-test-builder"
source "$top_dir"/tests/lib/fuzz_commonlib.sh

function pre_fun() {
    set_env "${test_name}" "$top_dir"
    make_fuzz_zip "$fuzz_file" "$fuzz_dir" "$test_dir"
    fuzz_zip=$(ls "$test_dir"/*fuzz.zip)
    if [[ -z "$fuzz_zip" ]]; then
        echo "fuzz zip file not found"
        exit 1
    fi
}

function test_fun() {
    local time=$1
    if [[ -z "$time" ]]; then
        time=1m
    fi
    go-fuzz -bin="$fuzz_zip" -workdir="$test_dir" &>> "$fuzz_log" &
    pid=$!
    if ! check_timeout $time $pid > /dev/null 2>&1; then
        echo "Can not kill process $pid"
    fi
    check_result "$fuzz_log"
    res=$?
    return $res
}

function main() {
    pre_fun
    test_fun "$1"
    res=$?
    if [ $res -ne 0 ];then
        exit 1
    else
        clean_env
    fi
}

main "$1"
