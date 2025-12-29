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

# Start a dummy web server in the background (port 8000)
python3 -m http.server 8000 > /dev/null 2>&1 &
SERVER_PID=$!
# kill at exit
trap 'kill $SERVER_PID' ERR

#breath
sleep 5

# Test if wrk is installed
which wrk || exit 1
# Test if /usr/bin/time is installed (required for resource measurement)
if [ ! -f /usr/bin/time ]; then echo "Error: /usr/bin/time not found. Please run init.sh to install 'time'."; exit 1; fi

# Run wrk
# Measure resources and duration, outputting to result.json
/usr/bin/time -f "{\"testcase\": \"$TESTCASE\", \"duration_sec\": %e, \"max_rss_kb\": %M, \"cpu_user_sec\": %U, \"cpu_sys_sec\": %S}" -o "${RESULT_DIR}/result.json" \
    /usr/bin/wrk -t12 -c400 -d10s http://127.0.0.1:8000/ > "${RESULT_DIR}/wrk.log" 2>&1

# create a success file to tell this test was ok
touch "${RESULT_DIR}/success"
