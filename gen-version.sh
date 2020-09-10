#!/bin/bash
###################################################################################################
# Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
# iSula-Kits licensed under the Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#     http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v2 for more details.
# Author: Xiang Li
# Create: 2020-05-18
# Description: This script used for update isula-build version and release. Enjoy and cherrs
###################################################################################################

# Basic info
top_dir=$(git rev-parse --show-toplevel)
version_file="${top_dir}/VERSION-openeuler"
spec_file="${top_dir}/isula-build.spec"
commit_file=${top_dir}/git-commit
color=$(tput setaf 2) # red
color_reset=$(tput sgr0)

# Commit ID
changeID=`git log -1 | grep Change-Id | awk '{print $2}' | head -c 40`
if [ "${changeID}" = "" ]; then
    changeID=`date | sha256sum | head -c 40`
fi
echo "${changeID}" > ${top_dir}/git-commit
commit_id=$(cat ${commit_file}|cut -c1-7)

old_all=$(cat "${version_file}")
old_version=$(cat "${version_file}" | awk -F"-" '{print $1}')
old_release=$(cat "${version_file}" | awk -F"-" '{print $2}')
major_old_version=$(echo "${old_version}" | awk -F "." '{print $1}')
minor_old_version=$(echo "${old_version}" | awk -F "." '{print $2}')
revision_old_version=$(echo "${old_version}" | awk -F "." '{print $3}')


# Read user input
read -rp "update version: Major(1), Minor(2), Revision(3), Release(4) [1/2/3/4]: " input
case ${input} in
    1)
        major_old_version=$((major_old_version + 1))
        minor_old_version="0"
        revision_old_version="0"
        new_release_num="1"
        ;;
    2)
        minor_old_version=$((minor_old_version + 1))
        revision_old_version="0"
        new_release_num="1"
        ;;
    3)
        revision_old_version=$((revision_old_version + 1))
        new_release_num="1"
        ;;
    4)
        new_release_num=$((old_release + 1))
        ;;

    *)
        echo "Wrong input, Version Not modified: ${old_version}"
        exit 0
        ;;
esac


# VERSION format:
# Major.Minor.Revision
new_version=${major_old_version}.${minor_old_version}.${revision_old_version}
new_release="${new_release_num}"
new_all=${new_version}-${new_release_num}

# Replace version and release for spec and VERSION files
sed -i -e "s/^Version: .*$/Version: ${new_version}/g" "${spec_file}"
sed -i -e "s/^Release: .*$/Release: ${new_release}/g" "${spec_file}"
echo "${new_all}" > "${version_file}"

if [[ "${old_all}" != "${new_all}" ]]; then
    printf 'Version: %s -> %s\n' "${old_all}" "${color}${new_all}${color_reset}"
fi

