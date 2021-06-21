#! /bin/bash

# Copyright © 2021 The OpenEBS Authors
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

set -e

## NOTE: This test should ran from openebs/dynamic-nfs-provisioner directory

echo "Running non-root application test"
cd ./ci/non-root-test
./non-root-test.sh
rc=$?
if [ $rc -ne 0 ]; then
    echo "Non-root application test failed"
    exit 1
fi
cd -

