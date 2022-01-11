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
# Create: 2021-05-29
# Description: test set new run and data root in configuration.toml
# History: 2021-8-18 Xiang Li <lixiang172@huawei.com> use busyobx instead of openeuler base image to speed up test
# History: 2022-01-10 Weizheng Xing <xingweizheng@huawei.com> Refactor: use systemd_run_command common function

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

run_root="/var/run/new-isula-build"
data_root="/var/lib/new-isula-build"
config_file="/etc/isula-build/configuration.toml"
image="hub.oepkgs.net/library/busybox:latest"

function clean() {
    rm -f $config_file
    mv "$config_file".bak $config_file

    systemctl restart isula-build
    rm -rf $run_root $data_root
}

# change to new data and run root
function pre_test() {
    cp $config_file "$config_file".bak
    sed -i "/run_root/d;/data_root/d" $config_file
    echo "run_root = \"${run_root}\"" >>$config_file
    echo "data_root = \"${data_root}\"" >>$config_file

    systemctl restart isula-build
}

# check if new resources are downloaded in new root
function do_test() {
    tree_node_befor=$(tree -L 3 $data_root | wc -l)
    systemd_run_command "isula-build ctr-img pull $image"
    tree_node_after=$(tree -L 3 $data_root | wc -l)

    check_value $((tree_node_after - tree_node_befor)) 8
    systemd_run_command "isula-build ctr-img rm $image"
}

pre_test
do_test
clean
exit "$exit_flag"
