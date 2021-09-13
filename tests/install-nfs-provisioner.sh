#!/bin/bash

# Copyright 2021 The OpenEBS Authors.
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

on_exit() {

    ## Changing back to original value
    sed -i '/OPENEBS_IO_ENABLE_ANALYTICS/!b;n;c\          value: "true"' deploy/kubectl/openebs-nfs-provisioner.yaml
}

trap 'on_exit' EXIT

sed -i 's/#- name: BackendStorageClass/- name: BackendStorageClass/' deploy/kubectl/openebs-nfs-provisioner.yaml
sed -i 's/#  value: "openebs-hostpath"/  value: "openebs-hostpath"/' deploy/kubectl/openebs-nfs-provisioner.yaml
## !b negates the previous address and breaks out of any processing, end the sed commands,
## n prints the current line and reads the next line into pattern space,
## c changes the current line to the string following command
sed -i '/OPENEBS_IO_ENABLE_ANALYTICS/!b;n;c\          value: "false"' deploy/kubectl/openebs-nfs-provisioner.yaml
kubectl  apply -f deploy/kubectl/openebs-nfs-provisioner.yaml

function waitForDeployment() {
	DEPLOY=$1
	NS=$2
	CREATE=true

	if [ $# -eq 3 ] && ! $3 ; then
		CREATE=false
	fi

	for i in $(seq 1 50) ; do
		kubectl get deployment -n ${NS} ${DEPLOY}
		kstat=$?
		if [ $kstat -ne 0 ] && ! $CREATE ; then
			return
		elif [ $kstat -eq 0 ] && ! $CREATE; then
			sleep 3
			continue
		fi

		replicas=$(kubectl get deployment -n ${NS} ${DEPLOY} -o jsonpath='{.status.readyReplicas}')
		if [ "$replicas" == "1" ]; then
			break
		else
			echo "Waiting for ${DEPLOY} to be ready"
			kubectl logs deploy/${DEPLOY} -n ${NS}
			sleep 10
		fi
	done
}

waitForDeployment openebs-nfs-provisioner openebs
kubectl get sc -o yaml
