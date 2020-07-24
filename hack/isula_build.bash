#!/usr/bin/env bash

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
# Create: 2020-07-01
# Description: bash completion file for isula-build commands

# To enable the completions, place this file in "/etc/bash_completion.d/"
# or just "source isula-build.bash" for temporary use


# completion for "isula-build"
_isula_build() {
    local index=1 cmd
    local first_class_commands="ctr-img login logout help version"
    local global_flags="--version --help --debug -log-level --timeout"

    while [[ "${index}" -lt "${COMP_CWORD}" ]]; do
        local s="${COMP_WORDS[index]}"
        case "${s}" in
            -*) ;;
            *)
                cmd="${s}"
                break
                ;;
        esac
        (( index++ ))
    done

    if [[ "${index}" -eq "${COMP_CWORD}" ]]; then
        local cur="${COMP_WORDS[COMP_CWORD]}"
        COMPREPLY=($(compgen -W "${first_class_commands} ${global_flags}" -- "${cur}"))
        return
    fi


    case "${cmd}" in
        ctr-img) _isula_build_ctr_img ;;
        login) _isula_build_login ;;
        logout) _isula_build_logout ;;
    esac
}

# complete for command "ctr-img"
_isula_build_ctr_img() {
    local index=1 subcommand_index
    while [[ "${index}" -lt ${COMP_CWORD} ]]; do
        local s="${COMP_WORDS[index]}"
        case "${s}" in
            ctr-img)
                subcommand_index=${index}
                break
                ;;
        esac
        (( index++ ))
    done

    while [[ ${subcommand_index} -lt ${COMP_CWORD} ]]; do
        local s="${COMP_WORDS[subcommand_index]}"
        case "${s}" in
            build)
                _isula_build_ctr_img_build
                return
                ;;
            images)
                _isula_build_ctr_img_images
                return
                ;;
            rm)
                _isula_build_ctr_img_rm
                return
                ;;
            help|-h)
                COMPREPLY=()
                return
                ;;
        esac
        (( subcommand_index++ ))
    done
    local cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=($(compgen -W "build images rm help" -- "${cur}"))
}

_isula_build_ctr_img_build() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    case ${prev} in
        '--filename'|'--iidfile')
            COMPREPLY=($(compgen -f -- "${cur}"))
            return
            ;;
        '--build-arg'|'--build-static'|'--output'|'--proxy'| '--help')
            return
            ;;
    esac
    COMPREPLY=($(compgen -W "--build-arg --build-static --filename --iidfile --output --proxy --help" -- "${cur}" ))
}

_isula_build_ctr_img_rm() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    case ${prev} in
        -*) return ;;
    esac
    COMPREPLY=($(compgen -W "--all --help" -- "${cur}" ))
}

_isula_build_ctr_img_images() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    case ${prev} in
        -*) return ;;
    esac
    COMPREPLY=($(compgen -W "--help" -- "${cur}" ))
}

_isula_build_login() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=($(compgen -W "--username --password-stdin --help" -- "${cur}"))
}

_isula_build_logout() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    case ${prev} in
        -*) return ;;
    esac
    COMPREPLY=($(compgen -W "--all --help" -- "${cur}"))
}


# completion for "isula-builder"
_isula_builder() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    COMPREPLY=()

    case ${prev} in
        '-v'|'--version'|'-h'|'--help'|'-D'|'--debug')
            return 0
            ;;
        '-c'|'--config')
            COMPREPLY=( $(compgen -f -- "${cur}")  )
            return 0
            ;;
        '--dataroot'|'--runroot')
            COMPREPLY=( $(compgen -d -- "${cur}") )
            return 0
            ;;
    esac

    local OPTS="--config --dataroot --debug --help --log-level --runroot --storage-driver --storage-opt --version"
    COMPREPLY=( $(compgen -W "${OPTS[*]}" -- "${cur}") )
}

complete -F _isula_build isula-build
complete -F _isula_builder isula-builder
