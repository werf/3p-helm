#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

source scripts/util.sh

if LANG=C sed --help 2>&1 | grep -q GNU; then
  SED="sed"
elif which gsed &>/dev/null; then
  SED="gsed"
else
  echo "Failed to find GNU sed as sed or gsed. If you are on Mac: brew install gnu-sed." >&2
  exit 1
fi

kube::util::ensure-temp-dir

export HELM_NO_PLUGINS=1

mkdir -p ${KUBE_TEMP}/docs/helm ${KUBE_TEMP}/docs/man/man1 ${KUBE_TEMP}/scripts
bin/helm docs --dir ${KUBE_TEMP}/docs/helm
bin/helm docs --dir ${KUBE_TEMP}/docs/man/man1 --type man
bin/helm docs --dir ${KUBE_TEMP}/scripts --type bash


FILES=$(find ${KUBE_TEMP} -type f)

${SED} -i -e "s:${HOME}:~:" ${FILES}
ret=0
for i in ${FILES}; do
  diff -NauprB -I 'Auto generated' ${i} $(echo ${i} | ${SED} "s:${KUBE_TEMP}/::") || ret=$?
done
if [[ $ret -eq 0 ]]; then
  echo "helm docs up to date."
else
  echo "helm docs are out of date. Please run \"make docs\""
  exit 1
fi
