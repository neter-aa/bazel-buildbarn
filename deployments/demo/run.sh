#!/usr/bin/env bash

set -eux

# Golang architecture of the current system.
ARCH="$(uname | tr '[A-Z]' '[a-z]')_amd64_pure_stripped"
# Location where the Buildbarn source tree is stored.
BBB_SRC="$(pwd)/../.."

CURWD="$(pwd)"
trap 'kill $(jobs -p)' EXIT TERM INT

# Clean up data from previous run.
rm -rf runner worker
mkdir -p storage-ac storage-cas worker

# Launch frontend, scheduler, storage, browser and worker.
"${BBB_SRC}/bazel-bin/cmd/bbb_frontend/${ARCH}/bbb_frontend" \
    -blobstore-config frontend-worker-blobstore.conf \
    -scheduler 'local|localhost:8981' \
    -web.listen-address localhost:7980 &
"${BBB_SRC}/bazel-bin/cmd/bbb_scheduler/${ARCH}/bbb_scheduler" \
    -web.listen-address localhost:7981 &
"${BBB_SRC}/bazel-bin/cmd/bbb_storage/${ARCH}/bbb_storage" \
    -blobstore-config storage-blobstore.conf \
    -web.listen-address localhost:7982 &
(cd "${BBB_SRC}/cmd/bbb_browser" &&
 exec "${BBB_SRC}/bazel-bin/cmd/bbb_browser/${ARCH}/bbb_browser" \
    -blobstore-config "${CURWD}/frontend-worker-blobstore.conf" \
    -web.listen-address localhost:7983) &
(cd worker &&
 exec "${BBB_SRC}/bazel-bin/cmd/bbb_worker/${ARCH}/bbb_worker" \
    -blobstore-config "${CURWD}/frontend-worker-blobstore.conf" \
    -browser-url http://localhost:7983/ \
    -concurrency 4 \
    -runner "unix://${CURWD}/runner" \
    -scheduler localhost:8981 \
    -web.listen-address localhost:7984) &
(cd worker &&
 exec "${BBB_SRC}/bazel-bin/cmd/bbb_runner/${ARCH}/bbb_runner" \
    -listen-path "${CURWD}/runner") &

wait
