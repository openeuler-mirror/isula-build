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
# Create: 2022-01-25
# Description: test priority of data and run root with different configuration locations( binary > configuration.toml > storage.toml)

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

config_file="/etc/isula-build/configuration.toml"
storage_file="/etc/isula-build/storage.toml"
main_run_root="/tmp/run/main-isula-build"
main_data_root="/tmp/lib/main-isula-build"
storage_run_root="/tmp/run/storage-isula-build"
storage_data_root="/tmp/lib/storage-isula-build"

# change to new data and run root
function pre_test() {
    cp $config_file "$config_file".bak
    cp $config_file "$config_file".bak

    cp $storage_file "$storage_file".bak
    cp $storage_file "$storage_file".bak
}

function clean() {
    rm -f "$config_file"
    rm -f "$storage_file"

    mv $config_file.bak "$config_file"
    mv $storage_file.bak "$storage_file"
}

function main_set_storage_not_set() {
    sed -i "/run_root/d;/data_root/d" $config_file
    sed -i "/runroot/d;/graphroot/d" $storage_file
    echo "run_root=\"$main_run_root\"" >>$config_file
    echo "data_root=\"$main_data_root\"" >>$config_file

    check_root "$main_run_root" "$main_data_root"
}

function main_not_set_storage_set() {
    sed -i "/run_root/d;/data_root/d" $config_file
    sed -i "/runroot/d;/graphroot/d" $storage_file
    eval "sed -i '/\[storage\]/a\runroot=\"$storage_run_root\"' $storage_file"
    eval "sed -i '/\[storage\]/a\graphroot=\"$storage_data_root\"' $storage_file"

    check_root "$main_run_root" "$main_data_root"
}

function main_set_storage_set() {
    sed -i "/run_root/d;/data_root/d" $config_file
    sed -i "/runroot/d;/graphroot/d" $storage_file
    echo "run_root = \"$main_run_root}\"" >>$config_file
    echo "data_root = \"$main_data_root\"" >>$config_file
    eval "sed -i '/\[storage\]/a\runroot=\"$storage_run_root\"' $storage_file"
    eval "sed -i '/\[storage\]/a\graphroot=\"$storage_data_root\"' $storage_file"

    check_root "$main_run_root" "$main_data_root"
}

# run command and check its result
# $1 (run root)
# $1 (data root)
function check_root() {
    local -r run_root="$1"
    local -r data_root="$2"

    start_time=$(date '+%Y-%m-%d %H:%M:%S')
    systemctl restart isula-build

    run_check_result "journalctl -u isula-build --since \"$start_time\" --no-pager | grep $run_root" 0
    run_check_result "journalctl -u isula-build --since \"$start_time\" --no-pager | grep $data_root" 0
}

pre_test
main_set_storage_not_set
main_not_set_storage_set
main_set_storage_set
clean
exit "$exit_flag"
