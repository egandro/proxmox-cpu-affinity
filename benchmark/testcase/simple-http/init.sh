#!/bin/bash

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"
echo "Initializing testcase: $TESTCASE"

apt-get update
apt-get install -y wrk time

TESTCASE_DIR="/testcase"
TESTCASE_WORK_DIR="$TESTCASE_DIR/$TESTCASE/work"
mkdir -p "$TESTCASE_WORK_DIR"
cd "$TESTCASE_WORK_DIR" || exit 1
