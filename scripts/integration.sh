#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc
./scripts/install_tools.sh

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}
export CF_STACK=${CF_STACK:-cflinuxfs3}

UNCACHED_BUILDPACK_FILE=${UNCACHED_BUILDPACK_FILE:-""}
CACHED_BUILDPACK_FILE=${CACHED_BUILDPACK_FILE:-""}

cd src/*/integration

echo "Run Uncached Buildpack without miniconda tests"
BUILDPACK_FILE="$UNCACHED_BUILDPACK_FILE" \
  ginkgo -r -mod=vendor -compilers=1 --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES --slowSpecThreshold=60 -- --cached=false

echo "Run Uncached Buildpack miniconda tests"
BUILDPACK_FILE="$UNCACHED_BUILDPACK_FILE" \
  ginkgo -r -mod=vendor -compilers=1 --flakeAttempts=$GINKGO_ATTEMPTS -nodes 1 --slowSpecThreshold=60 -- --cached=false --miniconda=true

echo "Run Cached Buildpack without miniconda tests"
BUILDPACK_FILE="$CACHED_BUILDPACK_FILE" \
  ginkgo -r -mod=vendor -compilers=1 --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES --slowSpecThreshold=60 -- --cached

echo "Run Uncached Buildpack miniconda tests"
BUILDPACK_FILE="$UNCACHED_BUILDPACK_FILE" \
  ginkgo -r -mod=vendor -compilers=1 --flakeAttempts=$GINKGO_ATTEMPTS -nodes 1 --slowSpecThreshold=60 -- --cached=true --miniconda=true
