#!/usr/bin/env sh

set -e

export WORKBOX_DIR=public/3rdparty/workbox

rm -rf ${WORKBOX_DIR}
workbox copyLibraries build/3rdparty/

mkdir -p ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-sw.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-core.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-strategies.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-routing.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-navigation-preload.prod.js ${WORKBOX_DIR}
mv build/3rdparty/workbox-*/workbox-precaching.prod.js ${WORKBOX_DIR}
rm -rf build/3rdparty/workbox-*

# Go's //go:embed fails when the matched directory exists but contains no embeddable files.
# This script uses build/3rdparty as a staging area, so remove it when empty.
rmdir build/3rdparty 2>/dev/null || true
