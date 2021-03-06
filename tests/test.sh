#!/bin/bash

top_dir=$(git rev-parse --show-toplevel)

# base test
function base() {
    source "$top_dir"/tests/lib/common.sh
    pre_check
    start_isula_builder

    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): "
        if ! bash "$testfile"; then
            exit 1
        fi
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "test_*" -type f -print)

    cleanup
}

# go fuzz test
function fuzz() {
    failed=0
    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): " | tee -a ${top_dir}/tests/fuzz.log
        bash "$testfile" "$1" | tee -a ${top_dir}/tests/fuzz.log
        if [ $PIPESTATUS -ne 0 ]; then
            failed=1
        fi
        # delete tmp files to avoid "no space left" problem
        find /tmp -maxdepth 1 -iname "*fuzz*" -exec rm -rf {} \;
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "fuzz_*.sh" -type f -print)
    exit $failed
}

# base test
function integration() {
    source "$top_dir"/tests/lib/common.sh
    pre_check
    systemctl restart isula-build

    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): "
        if ! bash "$testfile"; then
            exit 1
        fi
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "test_integration*" -type f -print)
}

# main function to chose which kind of test
function main() {
    case "$1" in
        fuzz)
            fuzz "$2"
            ;;
        base)
            base
        ;;
        integration)
            integration
        ;;
        *)
            echo "Unknow test type."
            exit 1
        ;;
    esac
}

export "ISULABUILD_CLI_EXPERIMENTAL"="enabled"
main "$@"
