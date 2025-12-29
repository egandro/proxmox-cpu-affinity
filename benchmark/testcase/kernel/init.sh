#!/bin/bash

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"
echo "Initializing testcase: $TESTCASE"

apt-get update
apt-get install -y build-essential git bison flex libssl-dev libelf-dev time

TESTCASE_DIR="/testcase/"
mkdir -p "$TESTCASE_DIR"
cd "$TESTCASE_DIR" || exit 1

git clone --depth=1 https://github.com/torvalds/linux.git
cd linux || exit 1

make defconfig
make clean
