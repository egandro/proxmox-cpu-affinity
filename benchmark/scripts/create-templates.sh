#/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

${SCRIPTDIR}/template-debian.sh ${CREATE_TEMPLATE_FLAGS}
