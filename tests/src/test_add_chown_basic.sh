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
# Author: Danni Xia
# Create: 2020-08-27
# Description: dockerfile test add-chown-basic

top_dir=$(git rev-parse --show-toplevel)
source "$top_dir"/tests/lib/common.sh

image_name=add-chown-basic
context_dir="$top_dir"/tests/data/add-chown-basic
test_build_without_output "$image_name" "$context_dir"
test_build_with_docker_archive_output "$image_name" "$context_dir"
test_build_with_docker_daemon_output "$image_name" "$context_dir"
test_build_with_isulad_output "$image_name" "$context_dir"

echo "PASS"
