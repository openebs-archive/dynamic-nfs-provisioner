# How to set File Permissions for the shared filesystem

You can use the 'FilePermissions' key in the `cas.openebs.io/config` PersistentVolumeClaim annotation to modify the owner, group and file modes of the shared NFS filesystem.

The file permission changes are handled before the NFS server initializes. 

The `chown` and `chmod` commands are run with the `--recursive` flag.<br>
**NOTE:** The commands are run only if the existing values for the owner, the group or the file mode of the root of the shared directory do not match with the requested values. This is similar to the Kubernetes [fsGroupChangePolicy's "OnRootMismatch"](https://kubernetes.io/blog/2020/12/14/kubernetes-release-1.20-fsgroupchangepolicy-fsgrouppolicy/#allow-users-to-skip-recursive-permission-changes-on-mount).

Declare ownership and file mode change specfications using the UID, GID and mode keys (sample PersistentVolumeClaim below):<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**UID:** This is the owner's user ID. Only valid UIDs are usable.<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**GID:** This is the group ID of the owning group. Only valid GIDs are usable.<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**mode:** Use this to specify the filesystem permission modes. Both octal and alphabet inputs are accepted. E.g. "0755", "g+rw".<br><br>
All of the keys are optional.
```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: secure-vol
  annotations:   
    cas.openebs.io/config: |
      - name: FilePermissions
        data:
          UID: "1000"
          GID: "2000"
          mode: "0744"
spec:
  storageClassName: openebs-kernel-nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
```
>**Note:** The FilePermissions config may be used with the `cas.openebs.io/config` StorageClass annotation key as well. If the config option is present on both the StorageClass and the PersistentVolumeClaim, the PersistentVolumeClaim config takes precedence.

## FSGID/fsGroup

The 'FSGID' config key (StorageClass annotations) is **being deprecated and will be removed in future releases**. At present, the FSGID config can be used, but it cannot be used along with FilePermission data keys 'GID' and/or 'mode'.

You can use the FilePermissions data values to change the group ownership and set a SetGID bit. This will result in changes similar to setting the fsGroup key in the NFS server Pod's securityContext (same as the previously-used FSGID `cas.openebs.io/config` key).

The following FilePermissions data values will result in a similar effect in volume permissions as setting fsGroup as '2000' with fsGroupChangePolicy as 'OnRootMismatch'.

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: secure-vol
  annotations:
    cas.openebs.io/config: |
      - name: FilePermissions
        data:
          GID: "2000"
          mode: "g+s"
spec:
  storageClassName: openebs-kernel-nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
```