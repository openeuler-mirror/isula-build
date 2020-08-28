#!/bin/bash

top_dir=$(git rev-parse --show-toplevel)
source "$top_dir"/tests/lib/common.sh

pre_check
start_isula_builder

while IFS= read -r testfile; do
    echo -e "test $testfile:\c"
    if ! bash "$testfile"; then
      exit 1
    fi
done < <(find "$top_dir"/tests/src -maxdepth 1 -type f -print)

cleanup
