#!/bin/bash

top_dir=$(git rev-parse --show-toplevel)

# normal test
function normal() {
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
    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): "
        if ! bash "$testfile" "$1"; then
          exit 1
        fi
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "fuzz_*" -type f -print)
}

# main function to chose which kind of test
function main() {
    case "$1" in
        fuzz)
            fuzz "$2"
            ;;
        *)
            normal
            ;;
    esac
}

main "$@"
