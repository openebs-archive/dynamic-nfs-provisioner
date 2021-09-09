#! /bin/bash

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
# This script runs tests and generates a report file.

if [ $# -ne 2 ]; then
    echo "Please specify NFS PVC name and namespace"
    echo "$0 <nfs_pvc_name> <nfs_pv_name>"
    exit 1
fi
nfs_pvc_name=$1
nfs_pvc_namespace=$2
backend_pvc_name=""
backend_pvc_namespace=""

nfs_pv_name=$(kubectl get pvc "${nfs_pvc_name}" -n "${nfs_pvc_namespace}" -o jsonpath='{.spec.volumeName}')
if [ -z "$nfs_pv_name" ]; then
    echo "Unable to find NFS PV for PVC ${nfs_pvc_namespace}/${nfs_pvc_name}"
    exit 1
fi

all_pvc_namespaces=$(kubectl get pvc --all-namespaces -o jsonpath="{range .items[*]}{@.metadata.name};{@.metadata.namespace}:{end}")

for pvc_name_namespace in `echo $all_pvc_namespaces | tr ":" " "`; do
    arr=(${pvc_name_namespace//;/ })
    pvc_name=${arr[0]}
    pvc_namespace=${arr[1]}
    if [ "nfs-${nfs_pv_name}" == "${pvc_name}" ]; then
        backend_pvc_name="$pvc_name"
        backend_pvc_namespace="$pvc_namespace"
        break
    fi
done

if [ -z "$backend_pvc_name" ] || [ -z "$backend_pvc_namespace" ]; then
    echo "Looks like ${nfs_pvc_namespace}/${nfs_pvc_name} is not a NFS PVC... Not able to find backend PVC"
    exit 1
fi

backend_pv_name=$(kubectl get pvc "${backend_pvc_name}" -n "${backend_pvc_namespace}" -o jsonpath='{.spec.volumeName}')
echo ""
echo ""
echo "----------------------------------------------------------------"
echo "Backend PVC Name: ${backend_pvc_name}"
echo "Backend PVC Namespace: ${backend_pvc_namespace}"
echo "Backend PV Name: ${backend_pv_name}"
echo "NFS PV Name: ${nfs_pv_name}"
echo "NFS PVC Namespace/Name: ${nfs_pvc_namespace}/${nfs_pvc_name}"
