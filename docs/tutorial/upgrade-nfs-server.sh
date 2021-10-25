#!/bin/bash

# Copyright 2021 The OpenEBS Authors. All rights reserved.
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

set -e

print_help() {
cat << EOF
Usage:
    $0 <options> IMAGE_TAG

Example:
    $0 -n nfs-ns 0.7.1

IMAGE_TAG is required to execute this script.
By default, this script uses 'openebs' namespace to search nfs-server deployment.
If you have used different namespace for nfs-server deployment then you must provide
the namespace using option(-n).
EOF
}

# NFS_SERVER_NS represent the namespace for nfs server deployment
# By default, it is set to 'openebs'.
NFS_SERVER_NS=openebs

# IMAGE_TAG represent the version for nfs server image
IMAGE_TAG=

# list_deployment list nfs-server deployment in NFS_SERVER_NS namespace
list_deployment() {
    local -n deploy=$1
    deploy=$(kubectl get deployment -n ${NFS_SERVER_NS} -l openebs.io/nfs-server --no-headers -o custom-columns=:.metadata.name)
}

# upgrade_deployment patch the given deployment in NFS_SERVER_NS with image-tag IMAGE_TAG
upgrade_deployment() {
    deploymentName=$1

    existingImage=$(kubectl get deployment -n openebs ${deploymentName} -o jsonpath='{.spec.template.spec.containers[0].image}')

    repo=${existingImage%:*}
    newImage=${repo}:${IMAGE_TAG}
    patchJson="{\"spec\": {\"template\": {\"spec\": {\"containers\" : [{\"name\" : \"nfs-server\", \"image\" : \"${newImage}\"}]}}}}"

    kubectl patch deploy -n ${NFS_SERVER_NS} ${deploymentName} -p "${patchJson}" > /dev/null
    exitCode=$?
    if [ $exitCode -ne 0 ]; then
        echo "ERROR: Failed to patch ${deploymentName} exit code: $exitCode"
        exit
    fi

    rolloutStatus=$(kubectl rollout status -n ${NFS_SERVER_NS} deployment/${deploymentName})
    exitCode=$?
    if [[ ($exitCode -ne 0) || ! (${rolloutStatus} =~ "successfully rolled out") ]]; then
        echo "ERROR: Failed to rollout status for ${deploymentName} exit code: $exitCode"
        exit
    fi
}

# options followed by ':' needs an argument
# see `man getopt`
shortOpts=hn:

# store the output of getopt so that we can assign it to "$@" using set command
# since we are using "--options" in getopt, arguments are passed via -- "$@"
PARSED=$(getopt --options ${shortOpts} --name "$0" -- "$@")
if [[ $? -ne 0 ]]; then
    echo "invalid arguments"
    exit 1
fi
# assign arguments to "$@"
eval set -- "${PARSED}"

while [[ $# -gt 0 ]]
do
    case "$1" in
        -h)
            print_help
            exit 0
            ;;
        -n)
            shift
            NFS_SERVER_NS=$1
            echo $1 $NFS_SERVER_NS
            ;;
        ## argument without options is mentioned after '--'
        --)
            shift
            IMAGE_TAG=$1
            break
    esac
    shift   # Expose the next argument
done

[[ -z $IMAGE_TAG ]] && print_help && exit 0

deployment=
list_deployment deployment

for i in ${deployment}; do
    upgrade_deployment ${i}
    echo "Deployment ${NFS_SERVER_NS}/${i} updated with image tag ${IMAGE_TAG}"
done
