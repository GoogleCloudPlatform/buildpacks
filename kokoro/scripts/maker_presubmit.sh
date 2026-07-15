#!/bin/bash
set -ex
cd "${KOKORO_ARTIFACTS_DIR}/piper/google3"
if [[ -z "${TEST_TARGET}" ]]; then
  echo "TEST_TARGET env must be set."
  exit 1
fi
blaze test "//third_party/gcp_buildpacks/maker/acceptance/${TEST_TARGET}" --bes_keywords="kokoro"
