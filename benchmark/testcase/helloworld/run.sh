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

mkdir -p "$TESTCASE_DIR"
cd "$TESTCASE_DIR"

# Redirect all output (stdout & stderr) to a main log file
exec > "${RESULT_DIR}/testcase.log" 2>&1

echo "hello"
date

id

[ -f /etc/here ] && cat /etc/here

# Example: Create multiple log files
for i in {1..3}; do
    echo "Creating a pseudo log $i"
    echo "Log entry $i" > "${RESULT_DIR}/detail-${i}.txt"
done

# Measure resources and duration, outputting to result.json
/usr/bin/time -f "{\"testcase\": \"$TESTCASE\", \"duration_sec\": %e, \"max_rss_kb\": %M, \"cpu_user_sec\": %U, \"cpu_sys_sec\": %S}" -o "${RESULT_DIR}/result.json" \
    sleep 10

date
echo "done"

# create a success file to tell this test was ok
touch "${RESULT_DIR}/success"
