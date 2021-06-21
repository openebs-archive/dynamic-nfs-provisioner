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


on_exit() {
    echo "PVCs in the cluster"
    kubectl get pvc -A

    echo "Pods in the cluster"
    kubectl get po -A

    echo "Events in the cluster"
    kubectl get events -A
}

trap 'on_exit' EXIT

## This test will perform following operations
## 1. Create nfs-server and expose nfs-share volume with non-root(nfs-share permissions) permissions
## 2. Create an application with some random user ID & group ID.
## 3. Test accessing nfs-volume from an application it should through permission denied errors
## 4. Update an application with suplemental groups IDs as nfs-share permissions
## 5. Now nfs-volume should be accessible from an application.

echo "Create nfs storageclass with non-root permissions"
kubectl apply -f non-root-sc.yaml

echo "Create an application with non-root permissions"
kubectl apply -f busy-box-deployment.yaml

echo "Wait for availability of busy-box pod"
kubectl wait --for=condition=available --timeout=550s deployment/busy-box

echo "Waiting for busy-box pod to come into running state"
kubectl wait --for=condition=Ready pod -l app=busy-box --timeout=500s

## FIXME:
## Since we are using openebs localpv hostpath it creates a directory
## with (drwxrwsrwx) permissions. Since directory is accessable to everyone
## we can't test non-root application. To fix this we are updating directory
## permissions to (drwxrwsr-x) i.e only root & group access the directory
echo "Changing permissions of nfs-share volume others can't access"
pv_name=$(kubectl get pvc non-root-nfs-pvc -o jsonpath='{.spec.volumeName}')
nfs_server_name=$(kubectl get po -l openebs.io/nfs-server=nfs-"$pv_name" -n openebs -o jsonpath='{.items[0].metadata.name}')
kubectl exec "${nfs_server_name}" -n openebs -- chmod o=rx /nfsshare

pod_name=$(kubectl get po -l app=busy-box -ojsonpath='{.items[0].metadata.name}')
set +e
echo "Try to create file on nfs shared volume"
kubectl exec "${pod_name}" -n default -- touch /datadir/testvolume
rc=$?
if [ $rc -ne 1 ]; then
    echo "Non root application shouldn't access the nfs share volume but it is able to create file"
    exit 1
fi
set -e
echo "Contents in nfs-share volume"
kubectl exec "${pod_name}" -n default -- ls /datadir -lrth

echo "Patching busy-box deployment with correct suplemental group permissions"
kubectl patch deploy busy-box -p '{"spec":{"template":{"spec":{"securityContext": {"supplementalGroups": [120]}}}}}'
sleep 5

echo "Waiting for deployment to get rollout"
rollout_status=$(kubectl rollout status --namespace default deployment/busy-box)
rc=$?; if [[ ($rc -ne 0) || ! ($rollout_status =~ "successfully rolled out") ]];
then echo "ERROR: Failed to rollout status for 'busy-box' error: $rc"; exit 1; fi

echo "Waiting for busy-box pod to come into running state"
kubectl wait --for=condition=Ready pod -l app=busy-box --timeout=500s


pod_name=$(kubectl get po -l app=busy-box -ojsonpath='{.items[?(@.status.phase=="Running")].metadata.name}')
echo "Try to create file on nfs shared volume"
kubectl exec "${pod_name}" -n default -- touch /datadir/testvolume
rc=$?
if [ $rc -ne 0 ]; then
    echo "Non root application with correct suplemental groups should access the nfs share volume but it is unable to create file"
    exit 1
fi

echo "After setting suplemental groups contents in nfs-share volume"
kubectl exec "${pod_name}" -n default -- ls /datadir -lrth

## Delete busy-box deployment and PVC
kubectl delete -f busy-box-deployment.yaml
