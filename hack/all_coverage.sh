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
# Description: shell script for all coverage
# Note: use this file by typing make test-cover
#       Do not run this script directly

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" || exit; pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}"/merge_coverage.sh

unit_coverage=${PWD}/cover_unit_test_all.out
sdv_coverage=${PWD}/cover_sdv_test_all.out
output_file=${PWD}/cover_test_all

merge_cover "${output_file}" "${sdv_coverage}" "${unit_coverage}"
