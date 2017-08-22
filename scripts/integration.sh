#!/usr/bin/env bash
set -euo pipefail

export ROOT=$(dirname $(readlink -f ${BASH_SOURCE%/*}))
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  echo "Installing ginkgo"
  (cd "$ROOT/src/python/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi
if [ ! -f "$ROOT/.bin/buildpack-packager" ]; then
  echo "Installing buildpack-packager"
  (cd "$ROOT/src/python/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}

cd $ROOT/src/python/integration
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached=false
