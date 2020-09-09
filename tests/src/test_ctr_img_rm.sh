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
# Author: Jingxiao Lu
# Create: 2020-09-07
# Description: dockerfile test multi-stage-builds

nonexistent_image="foo:bar"
# rm an nonexistent image
isula-build ctr-img rm ${nonexistent_image}  > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "FAIL"
  exit 1
fi

echo "PASS"
