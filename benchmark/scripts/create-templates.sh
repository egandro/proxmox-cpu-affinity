#/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config

${SCRIPTDIR}/template-debian.sh ${CREATE_TEMPLATE_FLAGS}
