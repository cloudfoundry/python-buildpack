#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOTDIR}/scripts/.util/tools.sh"

function main() {
  local src
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"

  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"

  local stack
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"

  echo "Run Uncached Buildpack"
  runSpecs false

  echo "Run Cached Buildpack"
  runSpecs true
}

function runSpecs() {
  local cached
  cached="${1}"

  if [ -d "${src}/integration/serial_tests" ] && \
     [ -d "${src}/integration/parallel_tests" ]; then
    set +e

    CF_STACK="${CF_STACK:-"${stack}"}" \
    BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
      ginkgo \
        -r \
        -mod vendor \
        --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
        -nodes "${GINKGO_NODES:-3}" \
        --slowSpecThreshold 60 \
          "${src}/integration/parallel_tests" \
        -- --cached="${cached}"

    exit_status=$?

    CF_STACK="${CF_STACK:-"${stack}"}" \
    BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
      ginkgo \
        -r \
        -mod vendor \
        --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
        --slowSpecThreshold 60 \
          "${src}/integration/serial_tests" \
        -- --cached="${cached}"

    (( exit_status = exit_status || $? ))

    exit $exit_status

  else
    CF_STACK="${CF_STACK:-"${stack}"}" \
    BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
      ginkgo \
        -r \
        -mod vendor \
        --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
        -nodes "${GINKGO_NODES:-3}" \
        --slowSpecThreshold 60 \
          "${src}/integration" \
        -- --cached="${cached}"
  fi
  set -e
}

main "${@:-}"
