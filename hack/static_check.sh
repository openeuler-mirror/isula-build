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
# Description: shell script for static checking

workspace=$(cd "$(dirname "$0")" && cd .. && pwd)
config_file=".golangci.yml"
check_type=$1
export GO111MODULE=on

# check binary file golangci-lint and it's config exist
function pre() {
    # check golangci-lint exist
    lint=$(command -v golangci-lint) > /dev/null 2>&1
    if [ -z "${lint}" ]; then
        echo "Could not find binary golangci-lint"
        exit 1
    fi

    # check config exist
    config_path=${workspace}/${config_file}
    if [[ ! -f ${config_path} ]]; then
        echo "Could not find config file for golangci-lint"
        exit 1
    fi
}

# last: only do static check for the very latest commit
# all : do static check for the whole project
function run() {
    case ${check_type} in
        last)
            ${lint} run --modules-download-mode vendor
            ;;
        all)
            ${lint} run --new=false --new-from-rev=false --modules-download-mode vendor
            ;;
        *)
            return
            ;;
    esac
}

pre
run
