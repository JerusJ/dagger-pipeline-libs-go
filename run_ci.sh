#!/usr/bin/env bash

set -euo pipefail

CI_DIR="ci"
FLAGS="$@"

PIPELINE_FILE="pipeline"

main() {
    echo "--> Running Pipeline..."
    echo "Flags: '${FLAGS}'"

    RUN_CMD=""
    if which dagger >/dev/null; then
        echo "'dagger' Binary Exists, will run GUI pipeline."
        RUN_CMD="dagger run go run ."
    elif which go >/dev/null; then
        echo "'go' Binary Exists, will compile and run Golang pipeline from scratch."
        RUN_CMD="go run ."
    elif test -f "./ci/${PIPELINE_FILE}" >/dev/null; then
        echo "'pipeline' Binary Exists, we have neither 'dagger' nor 'go' in PATH, so we'll fall back to using the pre-compiled binary"
        RUN_CMD="./${PIPELINE_FILE}"
    else
        echo "ERROR: we do not have: 'dagger' or 'go' in PATH, and we do not have a pre-compiled binary at ${PIPELINE_FILE}. No way to run CI locally. Exiting."
        return 1
    fi
    pushd "${CI_DIR}"
        RUN_CMD_FLAGS="${RUN_CMD} ${FLAGS}"
        echo "CMD: ${RUN_CMD_FLAGS}"
        eval ${RUN_CMD_FLAGS}
    popd
    echo "Done."
}

main
