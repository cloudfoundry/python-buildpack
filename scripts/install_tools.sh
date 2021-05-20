#!/bin/bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOTDIR}/scripts/.util/tools.sh"

function main() {
  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"
  util::tools::jq::install --directory "${ROOTDIR}/.bin"
}

main "${@:-}"
