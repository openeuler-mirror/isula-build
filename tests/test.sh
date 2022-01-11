#!/bin/bash
# shellcheck disable=SC1091
top_dir=$(git rev-parse --show-toplevel)

# base test
function base() {
    source "$top_dir"/tests/lib/base_commonlib.sh
    pre_check
    create_tmp_dir
    start_isula_builder

    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): "
        if ! bash "$testfile"; then
            exit 1
        fi
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "isula_build_base_command.sh" -type f -print)

    cleanup
}

# go fuzz test
function fuzz() {
    failed=0
    while IFS= read -r testfile; do
        printf "%-45s" "test $(basename "$testfile"): " | tee -a "$top_dir"/tests/fuzz.log
        bash "$testfile" "$1" | tee -a "$top_dir"/tests/fuzz.log
        if [ $PIPESTATUS -ne 0 ]; then
            failed=1
        fi
        # delete tmp files to avoid "no space left" problem
        find /tmp -maxdepth 1 -iname "*fuzz*" -exec rm -rf {} \;
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "fuzz_*.sh" -type f -print)
    exit $failed
}

# integration test
function integration() {
    source "$top_dir"/tests/lib/integration_commonlib.sh
    create_tmp_dir
    pre_integration

    while IFS= read -r testfile; do
        printf "%-65s" "test $(basename "$testfile"): "
        if ! bash "$testfile"; then
            echo "FAIL"
            continue
        fi
        echo "PASS"
    done < <(find "$top_dir"/tests/src -maxdepth 1 -name "test_*" -type f -print)
    after_integration
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

export ISULABUILD_CLI_EXPERIMENTAL="enabled"

main "$@"
