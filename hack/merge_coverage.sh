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
# Description: merge coverage from input coverage files
# Note: Do not run this script directly

# Usage: merge_cover outputfile file1 file2 ... fileN
# Input: first: outputfile name
#        remaining: coverage files
function merge_cover() {
    output_file_name="$1"
    input_coverages=( "${@:2}" )

    output_coverage_file=${output_file_name}.out
    output_html_file=${output_file_name}.html
    output_merge_cover=${output_file_name}.merge
    grep -r -h -v "^mode:" "${input_coverages[@]}" | sort > "$output_merge_cover"
    current=""
    count=0
    echo "mode: set" > "$output_coverage_file"
    # read the cover report from merge_cover, convert it, write to final coverage
    while read -r line; do
        block=$(echo "$line" | cut -d ' ' -f1-2)
        num=$(echo "$line" | cut -d ' ' -f3)
        if [ "$current" == "" ]; then
            current=$block
            count=$num
        elif [ "$block" == "$current" ]; then
            count=$((count + num))
        else
            # if the sorted two lines are not in the same code block, write the statics result of last code block to the final coverage
            echo "$current" $count >> "${output_coverage_file}"
            current=$block
            count=$num
        fi
    done < "$output_merge_cover"
    rm -rf "${output_merge_cover}"

    # merge the results of last line to the final coverage
    if [ "$current" != "" ]; then
        echo "$current" "$count" >> "${output_coverage_file}"
    fi

    go tool cover -html="${output_coverage_file}" -o "$output_html_file"
}
