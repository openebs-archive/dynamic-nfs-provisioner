# Dynamic NFS Volume Provisioner

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

<p align="justify">
<strong>OpenEBS Dynamic NFS PV provisioner</strong> can be used to dynamically provision 
NFS Volumes using different kinds of block storage available on the Kubernetes nodes. 
<br>
<br>
</p>

This project is under active development. 

## Install

Install NFS Provisioner
```
kubectl apply -f deploy/kubectl/openebs-nfs-provisioner.yaml
```

Create a StorageClass with required backing storage class. Example:
```
#Sample storage classes for OpenEBS Local PV
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
        value: "openebs-hostpath"
provisioner: openebs.io/nfsrwx
reclaimPolicy: Delete
```

You can now use `openebs-rwx` storage class to create RWX volumes.

## Contributing

Head over to the [CONTRIBUTING.md](./CONTRIBUTING.md).

## Community, discussion, and support

Learn how to engage with the OpenEBS community on the [community page](https://github.com/openebs/openebs/tree/master/community).

You can reach the maintainers of this project at:

- [Kubernetes Slack](http://slack.k8s.io/) channels: 
      * [#openebs](https://kubernetes.slack.com/messages/openebs/)
      * [#openebs-dev](https://kubernetes.slack.com/messages/openebs-dev/)
- [Mailing List](https://lists.cncf.io/g/cncf-openebs-users)

### Code of conduct

Participation in the OpenEBS community is governed by the [CNCF Code of Conduct](CODE-OF-CONDUCT.md).

## Inspiration/Credit
- https://github.com/sjiveson/nfs-server-alpine
