#!/bin/bash

if [ -z "$1" ]; then
    echo "Error: TESTCASE is required."
    exit 1
fi

TESTCASE="$1"

apt-get update
apt-get install -y build-essential git bison flex libssl-dev libelf-dev time

