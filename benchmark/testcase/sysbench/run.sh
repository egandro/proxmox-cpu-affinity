#!/bin/bash

set -e

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"
TIMESTAMP=$(date +%s)
RESULT_DIR="/result/${TESTCASE}/${TIMESTAMP}"
TESTCASE_DIR="/testcase/"

# create a failed file when this script fails
trap 'touch "${RESULT_DIR}/failed"' ERR

mkdir -p "$RESULT_DIR"

TESTCASE_WORK_DIR="$TESTCASE_DIR/$TESTCASE/work"
mkdir -p "$TESTCASE_WORK_DIR"
cd "$TESTCASE_WORK_DIR" || exit 1

# Redirect all output (stdout & stderr) to a main log file
exec > "${RESULT_DIR}/testcase.log" 2>&1

# Test if sysbench is installed
which sysbench || exit 1
# Test if /usr/bin/time is installed (required for resource measurement)
if [ ! -f /usr/bin/time ]; then echo "Error: /usr/bin/time not found. Please run init.sh to install 'time'."; exit 1; fi

# Run sysbench
# Measure resources and duration, outputting to result.json
/usr/bin/time -f "{\"testcase\": \"$TESTCASE\", \"duration_sec\": %e, \"max_rss_kb\": %M, \"cpu_user_sec\": %U, \"cpu_sys_sec\": %S}" -o "${RESULT_DIR}/result.json" \
    sysbench cpu --cpu-max-prime=20000 --threads="$(nproc)" run > "${RESULT_DIR}/sysbench.log"

# Look for: events per second (higher is better).

# create a success file to tell this test was ok
touch "${RESULT_DIR}/success"
