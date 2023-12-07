#!/usr/bin/env bash
# Copyright 2018-2020 The OpenEBS Authors. All rights reserved.
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
#
# This script checks if any files are modified by tests like go fmt. 

set -e

# message to be displayed if test fails.
TEST_NAME=$1


if [[ `git diff --shortstat | wc -l | tr -d ' '` != 0 ]]; then 
  echo "Some files got changed after $1";printf "\n";git --no-pager diff;printf "\n"; exit 1;
fi
