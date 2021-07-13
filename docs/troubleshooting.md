
This document describes how to troubleshoot the known issues. If this doesn't help, you can file an issue on github (https://github.com/openebs/dynamic-nfs-provisioner/issues), or talk to us on the openebs slack community.

- [Troubleshooting](#troubleshooting)
    - [Application pod remains in ContainerCreating state](#application-pod-remains-in-containercreating-state)
        - [Missing nfs client package](#missing-nfs-client-package)
        - [Invalid BackendStorageClass](#invalid-backendstorageclass)
        - [DNS lookup error](#dns-lookup-error)
    - [Application not able to write to the volume](#application-not-able-to-write-to-the-volume)

## Application pod remains in ContainerCreating state
### Missing nfs client package
This may happen if the host machine doesn’t have the nfs client package installed then the Kubelet won’t be able to mount the nfs volume. You can confirm this issue by running command ``kubectl describe pods -n <NAMESPACE> <POD_NAME>``. Check for the similar events as mentioned below:

```
Events:
  Type     Reason            Age               From               Message
  ----     ------            ----              ----               -------
  Warning  FailedScheduling  38s               default-scheduler  0/1 nodes are available: 1 pod has unbound immediate PersistentVolumeClaims.
  Normal   Scheduled         36s               default-scheduler  Successfully assigned default/busybox-6cd54c66b8-nghw5 to 192.168.1.4
  Warning  FailedMount       5s (x7 over 36s)  kubelet            MountVolume.SetUp failed for volume "pvc-77e80aab-55e7-4e7e-ad27-b6ee674c8db8" : mount failed: exit status 32
Mounting command: mount
Mounting arguments: -t nfs 10.0.0.121:/ /var/lib/kubelet/pods/ab18bbc8-6f6d-4178-ab8f-60b2f635da91/volumes/kubernetes.io~nfs/pvc-77e80aab-55e7-4e7e-ad27-b6ee674c8db8
Output: mount: /var/lib/kubelet/pods/ab18bbc8-6f6d-4178-ab8f-60b2f635da91/volumes/kubernetes.io~nfs/pvc-77e80aab-55e7-4e7e-ad27-b6ee674c8db8: bad option; for several filesystems (e.g. nfs, cifs) you might need a /sbin/mount.<type> helper program.
```

To resolve this issue, install the nfs-client package on the host machine. Refer: https://github.com/openebs/dynamic-nfs-provisioner#prerequisites for more information.


### Invalid BackendStorageClass
If you have already installed the nfs-client package and you are still observing this issue then check if nfs-server pod is in Pending state or not. If nfs storageclass is configured with `BackendStorageClass` and `BackendStorageClass` is not available then nfs-provisioner won’t be able to create the backend PV for nfs volume. Due to this, nfs-server pod will remain in `Pending` state. To solve this issue, you can create the `BackendStorageClass` or use the default storageclass by removing `BackendStorageClass` from nfs storageclass.


### DNS lookup error
This could happen if the nfs provisioner is configured(by setting OPENEBS_IO_NFS_SERVER_USE_CLUSTERIP to `false`) to expose nfs pv using domain name instead of ip address.

To resolve this issue, check if dns pod is running or not. Refer https://github.com/openebs/dynamic-nfs-provisioner/issues/7 for more information.


## Application not able to write to the volume
This could happen if the application is running with a non-root user.

By default, nfs-share volume is accessible only by root users. Refer to the issue for more details.

To make it accessible for specific non-root users, follow the below-mentioned steps. There is already a PR to fix this issue. Until this PR gets merged, this is a workaround as of now.

execute the following patch command(backend storageclasss should be specified with fsType to support this change) on nfs-pvc deployment.

```
kubectl patch deploy nfs-pvc-ab34af92-c914-4afb-a25a-517ba2aa12bf -p '{"spec":{"template":{"spec":{"securityContext": {"fsGroup": 100, "fsGroupChangePolicy": "OnRootMismatch"}}}}}' -n openebs
```

Now nfs-share volume permission is updated to 100.
Add the value of NFS volume group  100 under supplementalGroups of your application deployment/sts.

```
kubectl patch deploy <DEPLOYMENT_NAME> -p '{"spec":{"template":{"spec":{"securityContext": {"supplementalGroups": [100]}}}}}'
```

