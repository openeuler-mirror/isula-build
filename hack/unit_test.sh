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
# Author: iSula Team
# Create: 2020-07-11
# Description: go test script
# Note: use this file by typing make unit-test or make unit-test-cover
#       Do not run this script directly

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" || exit; pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}"/merge_coverage.sh

export GO111MODULE=on
run_coverage=$1
covers_folder=${PWD}/covers
testlog=${PWD}"/unit_test_log"
exclude_pattern="gopkgs|api/services"
go_test_mod_method="-mod=vendor"
go_test_count_method="-count=1"
go_test_timeout_flag="-timeout=300s"
go_test_race_flag="-race"
go_test_covermode_flag="-covermode=atomic"
go_test_coverprofile_flag="-coverprofile=/dev/null"

function precheck() {
    if pgrep isula-builder > /dev/null 2>&1; then
        echo "isula-builder is already running, please stop it first"
        exit 1
    fi
}

function run_unit_test() {
    TEST_ARGS=""
    if [ -n "${TEST_REG}" ]; then
        TEST_ARGS+=" -args TEST_REG=${TEST_REG}"
    fi
    if [ -n "${SKIP_REG}" ]; then
        TEST_ARGS+=" -args SKIP_REG=${SKIP_REG}"
    fi
    echo "Testing with args ${TEST_ARGS}"

    rm -f "${testlog}"
    if [[ -n ${run_coverage} ]]; then
        mkdir -p "${covers_folder}"
    fi
    for package in $(go list "${go_test_mod_method}" ./... | grep -Ev "${exclude_pattern}"); do
        echo "Start to test: ${package}"
        if [[ -n ${run_coverage} ]]; then
            coverprofile_file="${covers_folder}/$(echo "${package}" | tr / -).cover"
            go_test_coverprofile_flag="-coverprofile=${coverprofile_file}"
            go_test_covermode_flag="-covermode=set"
            go_test_race_flag=""
        fi
        # TEST_ARGS is " -args SKIP_REG=foo", so no double quote for it
        # shellcheck disable=SC2086
        go test -v "${go_test_race_flag}" "${go_test_mod_method}" "${go_test_coverprofile_flag}" "${go_test_covermode_flag}" -coverpkg=${package} "${go_test_count_method}" "${go_test_timeout_flag}" "${package}" ${TEST_ARGS} >> "${testlog}"
    done

    if grep -E -- "--- FAIL:|^FAIL" "${testlog}"; then
        echo "Testing failed... Please check ${testlog}"
    fi
    tail -n 1 "${testlog}"

    rm -f "${testlog}"
}

function generate_unit_test_coverage() {
    if [[ -n ${run_coverage} ]]; then
        merge_cover "cover_unit_test_all" "${covers_folder}"
        rm -rf "${covers_folder}"
    fi
}

precheck
run_unit_test
generate_unit_test_coverage
