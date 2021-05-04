# Dynamic NFS Volume Provisioner

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

<p align="justify">
<strong>OpenEBS Dynamic NFS PV provisioner</strong> can be used to dynamically provision 
NFS Volumes using different kinds of block storage available on the Kubernetes nodes. 
<br>
<br>
</p>

This project is under active development. 

## Prerequisites ##

Please ensure that an NFS client is functioning on all nodes that will run a pod that mounts an `openebs-rwx` volume. 

Here's how to prepare an NFS client on some common Operating Systems:

- **Ubuntu, Debian:** Install the `nfs-common` package if not already installed.

- **MacOS:** Should work out of the box.

- **Windows:**
Ensure that the default NFS client is operating. To do this start PowerShell as Administrator, and run `Install-WindowsFeature NFS-Client` if it's a Windows server or `Enable-WindowsOptionalFeature -FeatureName ServicesForNFS-ClientOnly, ClientForNFS-Infrastructure -Online -NoRestart` if it's a Windows host with a Desktop environment.

- **Fedora, CentOS, RedHat**: Install the `nfs-utils` package if not already installed.

- **FreeBSD**: 
   1) Edit the /etc/rc.conf file by setting or appending `nfs_client_enable="YES"`
   2) Run `service nfsclient start`

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
