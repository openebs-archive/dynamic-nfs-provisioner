# Non Root Applications Accessing OpenEBS NFS Volume

## Intro

There are multiple cases where non-root applications need access to NFS volume. Few examples are:
- Applications that are mandatory to run as non-root users that consume NFS volume.
- Multiple pods running with different permissions might need to dump the logs on common shared(NFS) volume.

To support above use cases OpenEBS NFS Provisioner provides an option to configure permissions of NFS volume via StorageClass.


## How To Use?

Non-root applications can consume NFS volume by following two steps:

- Create a PersistentVolumeClaim by specifying appropriate permissions under the FilePermissions config key.
  The volumes provisioned for this PVC will have its group-owner set to 120, and its SetGID bit enabled ([read more](../tutorial/file-permissions.md)).

Sample PersistentVolumeClaim:  
```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: nfs-pvc
  annotations:
    cas.openebs.io/config: |
      - name: FilePermissions
        data:
          GID: "120"
          mode: "g+s"
spec:
  storageClassName: openebs-kernel-nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 5G
```

- Now create an application by specifying the group ID value(i.e 120) under supplemental groups.
  When supplemental groups are specified corresponding user will be part of the same group
  and it makes volume accessible.
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fio
spec:
  restartPolicy: Never
  securityContext:
    runAsUser: 75
    runAsGroup: 75
    supplementalGroups:
    - 120
  containers:
  - name: perfrunner
    image: openebs/tests-fio
    imagePullPolicy: IfNotPresent
    command: ["/bin/bash"]
    args: ["-c", "while true ;do sleep 50; done"]
    volumeMounts:
       - mountPath: /datadir-fio
         name: fio-vol
    tty: true
  volumes:
  - name: fio-vol
    persistentVolumeClaim:
      claimName: nfs-pvc
```


### How to debug permission denied error?

This might be caused due to backend volume might not be updated with permissions.
Following are the steps to find permissions configured on backend volume:

- Run the following command
  ```sh
  kubectl exec nfs-pvc-f5e31497-a366-4987-9359-4265119db839-5949d8894-f78gc bash -n openebs -- stat /nfsshare
  ```
  Access will be `0755` if permissions are not configured on backend volume
  ```sh
  File: nfsshare/
  Size: 4096      	Blocks: 8          IO Block: 4096   directory
  Device: fd00h/64768d	Inode: 2           Links: 3
  Access: (0755/drwxr-xr-x)  Uid: (    0/    root)   Gid: (    0/    root)
  Access: 2021-07-09 11:23:42.755974808 +0000
  Modify: 2021-07-09 11:23:41.933968717 +0000
  Change: 2021-07-09 11:23:41.933968717 +0000
  ```
