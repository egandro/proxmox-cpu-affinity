#/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

apt update -qq
apt install jq -qq -y
