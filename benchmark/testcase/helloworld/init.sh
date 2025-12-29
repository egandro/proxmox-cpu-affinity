#!/bin/bash

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"
echo "Initializing testcase: $TESTCASE"

apt-get update
apt-get install -y time

TESTCASE_DIR="/testcase/"
mkdir -p "$TESTCASE_DIR"
cd "$TESTCASE_DIR"
