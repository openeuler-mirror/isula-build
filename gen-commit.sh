#!/bin/sh

# Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
# Description: This shell script is used to generate commitID store file.
# Author: xiadanni1@huawei.com
# Create: 2020-07-20

changeID=`git log -1 | grep Change-Id | awk '{print $2}' | head -c 40`
if [ "${changeID}" = "" ]; then
    changeID=`date | sha256sum | head -c 40`
fi
echo "${changeID}" > git-commit 
