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
TESTCASE_WORK_DIR="$TESTCASE_DIR/$TESTCASE/work"
mkdir -p "$TESTCASE_WORK_DIR"
cd "$TESTCASE_WORK_DIR" || exit 1

rm -rf linux
VERSION=v6.18
git clone --depth=1 --branch "$VERSION" https://github.com/torvalds/linux.git
cd linux || exit 1

#git checkout

make defconfig || exit 1
make clean || exit 1
