#! /bin/bash

set -e

echo "Running non-root application test"
cd non-root-test
./non-root-test.sh
rc=$?
if [ $rc -ne 0 ]; then
    echo "Non-root application test failed"
    exit 1
fi
cd -

