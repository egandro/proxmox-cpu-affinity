#/bin/bash

set -e

SCRIPTDIR="$(dirname "$0")"

. ${SCRIPTDIR}/../config.sh

export VM_ID=${TEMPLATE_EMPTY_VM_ID}
${SCRIPTDIR}/template-empty.sh ${CREATE_TEMPLATE_FLAGS}

export VM_ID=${TEMPLATE_DEBIAN_VM_ID}
${SCRIPTDIR}/template-debian.sh ${CREATE_TEMPLATE_FLAGS}
