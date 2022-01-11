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
# Description: common functions for isula-build integration tests

top_dir=$(git rev-parse --show-toplevel)
# shellcheck disable=SC1091
source "$top_dir"/tests/lib/common.sh

run_root="/var/run/integration-isula-build"
data_root="/var/lib/integration-isula-build"
config_file="/etc/isula-build/configuration.toml"

function pre_integration() {
    rm -f "$TMPDIR"/buildlog-failed

    cp $config_file "$config_file".integration
    sed -i "/run_root/d;/data_root/d" $config_file
    echo "run_root = \"${run_root}\"" >>$config_file
    echo "data_root = \"${data_root}\"" >>$config_file

    systemctl restart isula-build
}

function after_integration() {
    systemd_run_command "isula-build ctr-img rm -a"

    rm -f $config_file
    mv "$config_file".integration $config_file
    systemctl stop isula-build
    rm -rf $run_root $data_root
}
