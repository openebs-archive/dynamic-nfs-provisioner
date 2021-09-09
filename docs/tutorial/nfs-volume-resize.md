# How to Expand/Resize NFS Volume

OpenEBS dynamic-nfs-provisioner **doesn't support** natively expanding NFS volume but we can accomplish the same operation by following a workaround.


For expanding NFS volume, you must ensure the following items(prerequisites) are taken care of:

- The BackendStorageClass must support volume expansion, which can configure by editing the StorageClass definition to have `allowVolumeExpansion: true`

  ```
  kubectl patch sc openebs-lvmpv -p '{"allowVolumeExpansion": true}'
  ```

  ```sh
  storageclass.storage.k8s.io/openebs-lvmpv patched
  ```
- NFS StorageClass must support volume expansion, which can configure by editing the NFS StorageClass defination to have `allowVolumeExpansion: true`

  ```
  kubectl patch sc openebs-rwx -p '{"allowVolumeExpansion": true}'
  ```

  ```sh
  storageclass.storage.k8s.io/openebs-rwx patched
  ```

## Steps to perform NFS Volume Expansion

- After meeting the prerequisites, Resize NFS volume by editing backend PVC `spec.resources.requests.storage` to reflect the newly desired size, which must be greater than the original size.
  ```sh
  kubectl patch pvc <backend-pvc-name> -n <openebs-namespace> -p '{"spec": {"resources": {"requests": {"storage": "10Gi"}}}}'
  ```
  **Note**: OpenEBS dynamic-nfs-provisioner will create backend PVC to provide shared volume and backend PVC has following naming convention `nfs-<application-pv-name>`. Below command will help to get backend PVC name
  ```sh
  kubectl get pvc -n <openebs-namespace> -l nfs.openebs.io/nfs-pvc-name=<nfs_pvc_name> -o jsonpath='{.items[0].metadata.name}'
  ```
- Wait till success expansion of backend PVC i.e `.status.capacity.storage` should match with `spec.resources.requests.storage`
  ```sh
  kubectl get pvc <backend-pvc-name> -n <openebs-namespace>
  ```
- Update the NFS PV capacity manually(Since resize is not supported natively by dynamic-nfs provisioner this step will be a workaround)
  ```sh
  kubectl patch pv <nfs-pv-name> -p '{"spec": {"capacity": {"storage": "2Gi"}}}'
  ```

### Example

Example to show how to expanding NFS volumes for an already existing StorageClass(backend & NFS StorageClass), you can edit the StorageClass to include the `allowVolumeExpansion: true` if it is not marked for allowVolumeExpansion.

- Below YAML shows detailed information about backend StorageClass
  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    creationTimestamp: "2021-09-09T08:21:34Z"
    name: openebs-lvmpv
  parameters:
    storage: lvm
    vgpattern: ^lvm$
  provisioner: local.csi.openebs.io
  reclaimPolicy: Delete
  volumeBindingMode: Immediate
  allowVolumeExpansion: true
  ```
- Below YAML shows detailed information about NFS StorageClass
  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: StorageClass
  metadata:
    annotations:
      cas.openebs.io/config: |
        - name: NFSServerType
          value: "kernel"
        - name: BackendStorageClass
          value: "openebs-lvmpv"
      openebs.io/cas-type: nfsrwx
    creationTimestamp: "2021-09-09T08:23:39Z"
  provisioner: openebs.io/nfsrwx
  reclaimPolicy: Delete
  volumeBindingMode: Immediate
  allowVolumeExpansion: true
  ```
- Below are the details of Wordpress application consuming `RWX` volume

  ```sh
  kubectl get pvc -n wordpress

  NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS    AGE
  wordpress-persistent-storage   Bound    pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72   1Gi        RWX            openebs-rwx     75m

  kubectl get po -n wordpress

  NAME                               READY   STATUS    RESTARTS   AGE
  wordpress-6db6fb5444-5vj52         1/1     Running   0          76m

  kubectl get pv

  NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                                  STORAGECLASS    REASON   AGE
  pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72   1Gi        RWX            Delete           Bound    wordpress/wordpress-persistent-storage                 openebs-rwx              77m
  ```
- Backend PVC details for above `RWX` volume
  ```sh
  kubectl get pvc -n openebs

  NAME                                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS    AGE
  nfs-pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72   Bound    pvc-2134156f-681e-456b-b6cb-802f754a420f   1Gi        RWO            openebs-lvmpv   85m
  ```
- To Resize the backend PVC, edit the PVC capacity `spec.resources.requests.storage` to 3Gi. It may take few seconds to update the actual size in PVC resource, wait for the updated capacity to reflect in PVC status(`pvc.status.capacity.storage`). We can look at events of PVC to know information about resize:
  ```sh
  kubectl patch pvc nfs-pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72 -n openebs -p '{"spec": {"resources": {"requests": {"storage": "3Gi"}}}}'

  persistentvolumeclaim/nfs-pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72 patched
  ```
  ```sh
  kubectl describe pvc -n openebs nfs-pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72
  ...
  ...
  Events:
  Type     Reason                      Age                From                                   Message
  ----     ------                      ----               ----                                   -------
  Normal   Resizing                    72s (x2 over 67m)  external-resizer local.csi.openebs.io  External resizer is resizing volume pvc-2134156f-681e-456b-b6cb-802f754a420f
  Warning  ExternalExpanding           72s (x2 over 67m)  volume_expand                          Ignoring the PVC: didn't find a plugin capable of expanding the volume; waiting for an external controller to process this PVC.
  Normal   FileSystemResizeRequired    72s (x2 over 67m)  external-resizer local.csi.openebs.io  Require file system resize of volume on node
  Normal   FileSystemResizeSuccessful  6s (x2 over 66m)   kubelet                                MountVolume.NodeExpandVolume succeeded for volume "pvc-2134156f-681e-456b-b6cb-802f754a420f"
  ```
- Update the NFS PV capacity by running following command
  ```sh
  kubectl patch pv pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72 -p '{"spec": {"capacity": {"storage": "3Gi"}}}'
  ```
- Now, exec into the wordpress application pod(RWX volume consumer) and check the mount point Available space
  ```sh
  root@wordpress-6db6fb5444-5vj52:/var/www/html# df -h
  Filesystem      Size  Used Avail Use% Mounted on
  overlay         916G   71G  798G   9% /
  tmpfs            64M     0   64M   0% /dev
  tmpfs           7.8G     0  7.8G   0% /sys/fs/cgroup
  /dev/sda2       916G   71G  798G   9% /etc/hosts
  shm              64M     0   64M   0% /dev/shm
  ----------------------------------------------------------------------
  10.0.0.180:/    3.0G   29M  2.9G   1% /var/www/html                   |
  ----------------------------------------------------------------------
  tmpfs            14G   12K   14G   1% /run/secrets/kubernetes.io/serviceaccount
  ```

  Above output shows NFS volume has successfully expanded.

<br></br>

**Gotcha**: NFS PVC will still report old capacity, once dynamic-nfs-provisioner shifts towards CSI it can support dynamically expanding RWX volume.

<br></br>

#### Tip

- Download and run [script](./get-nfs-volume-details.sh) by passing NFS PVC name & namespace as input arguments to test and
  it will display backend PVC & PV details for given NFS PVC
  ```sh
  ./get-nfs-volume-details.sh wordpress-persistent-storage wordpress

  ----------------------------------------------------------------
  Backend PVC Name: nfs-pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72
  Backend PVC Namespace: openebs
  Backend PV Name: pvc-2134156f-681e-456b-b6cb-802f754a420f
  NFS PV Name: pvc-5a8bb1f2-c183-44a7-aa70-12f3138e2a72
  NFS PVC Namespace/Name: wordpress/wordpress-persistent-storage
  ```
