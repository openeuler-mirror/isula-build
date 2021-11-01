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
# Description: common function for save/load separated image

exit_flag=0

# $1: from image name
# $2: build image name
# $3: layers number
# $4: Dockerfile path
function touch_dockerfile() {
    cat > "$4" << EOF
FROM $1
MAINTAINER DCCooper
EOF
    for i in $(seq "$3"); do
        echo "RUN echo \"This is $2 layer ${i}: ${RANDOM}\" > line.${i}" >> "$4"
    done
}

# $1: from image name
# $2: build image name
# $3: layers number
# $4: Dockerfile path
function touch_bad_dockerfile() {
    cat > "$4" << EOF
FROM $1
MAINTAINER DCCooper
EOF
    for i in $(seq "$3"); do
        echo "RUN echo \"This is $2 layer ${i}: ${RANDOM}\"" >> "$4"
    done
}

# $1: image name
# $2: context dir
function build_image() {
    isula-build ctr-img build -t "$1" "$2"
}

function touch_rename_json() {
    cat > "$1" << EOF
[
    {
        "name": "app1_latest_app_image.tar.gz",
        "rename": "app1.tar.gz"
    },
    {
        "name": "app2_latest_app_image.tar.gz",
        "rename": "app2.tar.gz"
    },
    {
        "name": "app1_latest_base_image.tar.gz",
        "rename": "base1.tar.gz"
    },
    {
        "name": "app2_latest_base_image.tar.gz",
        "rename": "base2.tar.gz"
    },
    {
        "name": "app1_latest_lib_image.tar.gz",
        "rename": "lib1.tar.gz"
    },
    {
        "name": "app2_latest_lib_image.tar.gz",
        "rename": "lib2.tar.gz"
    }
]
EOF
}

function touch_bad_rename_json() {
    touch_rename_json "$1"
    sed -i '2d' "$1"
}

function check_result_equal() {
    if [[ $1 -eq $2 ]]; then
        return 0
    else
        ((exit_flag++))
        return 1
    fi
}

function check_result_not_equal() {
    if [[ $1 -ne $2 ]]; then
        return 0
    else
        ((exit_flag++))
        return 1
    fi
}
