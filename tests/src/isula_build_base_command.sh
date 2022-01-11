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
# Create: 2021-01-12
# Description: test isula-build base commands

top_dir=$(git rev-parse --show-toplevel)
source "$top_dir"/tests/lib/base_commonlib.sh

function test_isula_build_output_with_different_dockerfiles() {
    declare -A dockerfiles=(
        ["build-from-scratch"]="$top_dir"/tests/data/build-from-scratch
        ["add-chown-basic"]="$top_dir"/tests/data/add-chown-basic
        ["multi-files-env"]="$top_dir"/tests/data/multi-files-env
        ["multi-stage-builds"]="$top_dir"/tests/data/multi-stage-builds
    )

    for image_name in "${!dockerfiles[@]}"; do
        printf "%-45s" "Outputs with dockerfile $image_name:"
        test_isula_build_output "$image_name" "${dockerfiles[$image_name]}"
        echo "PASS"
    done
}

function test_isula_build_all_base_command() {
    image_name=build-from-scratch
    context_dir="$top_dir"/tests/data/build-from-scratch

    test_isula_build_base_command "$image_name" "$context_dir"
    echo "PASS"
}

echo ""
test_isula_build_output_with_different_dockerfiles
test_isula_build_all_base_command
