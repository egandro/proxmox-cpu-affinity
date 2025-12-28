#/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

if [ -z "$ORCHESTRATOR_MODE" ] && [ -f "${SCRIPTDIR}/../.env" ]; then
    . "${SCRIPTDIR}/../.env"
fi

apt update -qq
apt install jq python3-dotenv -qq -y
