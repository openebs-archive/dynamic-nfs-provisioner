# NFS Server Node Affinity

There are various use cases to assign pods to a specific set of nodes based on their behavior. In big environment, there will set of nodes dedicated to storage. Storage specific pod instances will be running only storage nodes. To enforce affinity rules Kubernetes provides various mechanisms among then OpenEBS NFS provisioner makes use of `requiredDuringSchedulingIgnoredDuringExecution` affinity[Which is best fit for this use case].

**How to use node affinity**?

- Label eligible nodes if not labeled with specific information related to NFS storage
  ```sh
  kubectl label node <node-1> <node-2> openebs.io/storage=true openebs.io/nfs-server=true
  ```

- Deploy NFS Provisioner by specifying `OPENEBS_IO_NFS_SERVER_NODE_AFFINITY` env with
  corresponding value
  ```sh
  - name: OPENEBS_IO_NFS_SERVER_NODE_AFFINITY
    value: openebs.io/storage,openebs.io/nfs-server
  ```
  Note: It also supports taking values for labels as an example value can be
        `kubernetes.io/zone:[zone1,zone2],openebs.io/storage,openebs.io/nfs-server`

- Deploy NFS StorageClass as shown below
  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    name: openebs-rwx
    annotations:
      openebs.io/cas-type: nfsrwx
      cas.openebs.io/config: |
        - name: NFSServerType
          value: "kernel"
        - name: BackendStorageClass
          value: "openebs-lvm-localpv"
  provisioner: openebs.io/nfsrwx
  reclaimPolicy: Delete
  ```

- Deploy PVC refering to above storage class
  ```yaml
  kind: PersistentVolumeClaim
  apiVersion: v1
  metadata:
    name: nfs-pvc
  spec:
    storageClassName: openebs-rwx
    accessModes:
      - ReadWriteMany
    resources:
      requests:
        storage: 5G
  ```

- Verify whether NFS Server instance is scheduled on desired nodes
  ```sh
  kubectl get po -l openebs.io/nfs-server -n openebs -o wide
  ```
