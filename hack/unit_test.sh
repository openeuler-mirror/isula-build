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

TEST_ARGS=""
if [ ! -z "${TEST_REG}" ]; then
    TEST_ARGS+=" -args TEST_REG=${TEST_REG}"
fi
if [ ! -z "${SKIP_REG}" ]; then
    TEST_ARGS+=" -args SKIP_REG=${SKIP_REG}"
fi
echo "Testing with args ${TEST_ARGS}"

testlog=${PWD}"/unit_test_log"
rm -f "${testlog}"
touch "${testlog}"
for path in $(go list ./...); do
    echo "Start to test: ${path}"
    # TEST_ARGS is " -args SKIP_REG=foo", so no double quote for it
    go test -mod=vendor -cover -count=1 -timeout 300s -v "${path}" ${TEST_ARGS} >> "${testlog}"
    cat "${testlog}" | grep -E -- "--- FAIL:"
    if [ $? -eq 0 ]; then
        echo "Testing failed... Please check ${testlog}"
        exit 1
    fi
    tail -n 1 "${testlog}"
done

rm -f "${testlog}"
