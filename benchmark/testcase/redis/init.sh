#!/bin/bash

set -e
set -x

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"
echo "Initializing testcase: $TESTCASE"

apt-get update
# yes redis needs tcl for tests...
apt-get install -y time git pkg-config tcl build-essential

TESTCASE_DIR="/testcase/"
TESTCASE_WORK_DIR="$TESTCASE_DIR/$TESTCASE/work"
mkdir -p "$TESTCASE_WORK_DIR"
cd "$TESTCASE_WORK_DIR" || exit 1

rm -rf redis
VERSION=8.4.0
git clone https://github.com/redis/redis.git
cd redis || exit 1
git checkout $VERSION  || exit 1

make clean
