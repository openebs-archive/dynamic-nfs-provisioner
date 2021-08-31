# Configure NFS Server Resource Requirements

Resource requirements([requests & limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#requests-and-limits)) are required to improve the quality of service of a process. For more information about resource requirements refer following docs:
- [CPU requests & limits](https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#motivation-for-cpu-requests-and-limits)
- [Memory requests & limits](https://kubernetes.io/docs/tasks/configure-pod-container/assign-memory-resource/#motivation-for-memory-requests-and-limits)


**How to configure NFS server with resource requirements**?

- Create a NFS StorageClass by specifying resource requests & limits under `.metadata.annotations` as shown below:
  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    name: openebs-rwx-resource-req
    annotations:
      openebs.io/cas-type: nfsrwx
      cas.openebs.io/config: |
        - name: NFSServerType
          value: kernel
        - name: BackendStorageClass
          value: openebs-hostpath
        - name: NFSServerResourceRequests
          value: |-
            cpu: 50m
            memory: 50Mi
        - name: NFSServerResourceLimits
          value: |-
            cpu: 100m
            memory: 100Mi
  provisioner: openebs.io/nfsrwx
  reclaimPolicy: Delete
  ```
  save above YAML into `openebs-rwx-resource-req.yaml` and create StorageClass in the cluster using below command:
  ```sh
  kubectl apply -f openebs-rwx-resource-req.yaml
  ```
- Create a NFS PVC refering to above StorageClass
  ```yaml
  kind: PersistentVolumeClaim
  apiVersion: v1
  metadata:
    name: nfs-pvc
  spec:
    storageClassName: openebs-rwx-resource-req
    accessModes:
      - ReadWriteMany
    resources:
      requests:
        storage: 5G
  ```
  save above YAML into `nfs-pvc.yaml` and create PersistentVolumeClaim in the cluster using below command:
  ```sh
  kubectl apply -f nfs-pvc.yaml
  ```
- Check for NFS server pod
  ```sh
  kubectl get po -n openebs

  NAME                                                            READY   STATUS    RESTARTS   AGE
  nfs-pvc-a970946b-aad5-47b3-93ff-ac238691dce0-5d9df94974-x6zxk   1/1     Running   0          38s
  openebs-localpv-provisioner-b8d8d6967-8pxf7                     1/1     Running   0          25m
  openebs-nfs-provisioner-778b7f46d9-gslwv                        1/1     Running   0          16m
  ```
- Verify whether NFS server pod contains resource requirements(requests & limits)
  ```sh
  kubectl get po nfs-pvc-a970946b-aad5-47b3-93ff-ac238691dce0-5d9df94974-x6zxk -n openebs -ojsonpath='{.spec.containers[0].resources}'

  {"limits":{"cpu":"100m","memory":"100Mi"},"requests":{"cpu":"50m","memory":"50Mi"}}
  ```
