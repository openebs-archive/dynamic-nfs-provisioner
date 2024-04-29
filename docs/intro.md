# QuickStart
## Prerequisites
Before installing nfs-provisioner make sure your Kubernetes cluster meets the following prerequisites:

1. Kubernetes version 1.18
2. This guide assumes you have a backend storageclass available, either a default one or any other (e.g: https://openebs.io/docs/user-guides/localpv-hostpath).
3. NFS Client is installed on all nodes that will run a pod that mounts an `openebs-rwx` volume.
   Here's how to prepare an NFS client on some common Operating Systems:

| OPERATING SYSTEM |  How to install NFS Client package                                |
| ---------------- | -------------------------------------------------------- |
| RHEL/CentOS/Fedora  |run *sudo yum install nfs-utils -y*      |
| Ubuntu/Debian   |run *sudo apt install nfs-common -y*     |
| MacOS     |Should work out of the box |
| FreeBSD  |Edit the */etc/rc.conf* file by setting or appending *nfs_client_enable="YES"* |
| Windows   |1. Start PowerShell as Administrator.<br/>2. In case of Windows server, run *Install-WindowsFeature NFS-Client*<br/>3. In case of Windows host with a Desktop environment, run *Enable-WindowsOptionalFeature -FeatureName ServicesForNFS-ClientOnly, ClientForNFS-Infrastructure -Online -NoRestart* |

## Install
### Install NFS Provisioner through kubectl
To install NFS Provisioner through kubectl, run below command:
```
kubectl apply -f https://openebs-archive.github.io/charts/nfs-operator.yaml
```

Above command will install the NFS Provisioner in *openebs* namespace and creates a Storageclass named *openebs-rwx*, with backend Storageclass *openebs-hostpath*.

Above installation will use latest stable release tag. To install a specific release version of nfs-provisioner, you can download YAML file from [here](https://github.com/openebs/charts/tree/gh-pages).


### Install NFS Provisioner through Helm
You can install NFS Provisioner through helm using below command:

```
helm repo add openebs https://openebs-archive.github.io/charts
helm repo update
helm install openebs openebs/openebs -n openebs --create-namespace --set nfs-provisioner.enabled=true
```

<details>
  <summary>Click here for configuration options.</summary>

  1. Install OpenEBS NFS Provisioner without NDM and Dynamic LocalPV Provisioner.

     You may choose to exclude the NDM and LocalPV subchart from installation if...
     - you want to only use OpenEBS NFS Provisioner
     - you already have NDM and LocalPV installed. Check if
        - NDM pods exist with the command `kubectl get pods -n openebs -l 'openebs.io/component-name in (ndm, ndm-operator)'`
        - LocalPV pods exists with the command `kubectl get pods -n openebs -l 'openebs.io/component-name in (openebs-localpv-provisioner)'`

```console
helm install openebs openebs/openebs -n openebs --create-namespace \
  --set ndm.enabled=false \
  --set ndmOperator.enabled=false \
  --set localprovisioner.enabled=false  \
  --set nfs-provisioner.enabled=true
```
</details>

[Click here](https://github.com/openebs/dynamic-nfs-provisioner/tree/develop/deploy/helm/charts) for detailed instructions on using the Helm chart.

## Provision NFS Volume
To provision NFS Volume, NFS Provisioner creates the following resources:

    - Create Backend PVC by using BackendStorageclass, mentioned in NFS Storageclass, or default Storageclass.
    - Deploy NFS-Server using Deployment which will mount the PV created by Backend PVC
    - Create Service to expose NFS-Server
    - Create PV with NFS mount information

To provision NFS Volume, create a NFS Storageclass with required backend Storageclass. Sample Storageclass YAML is as below:

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
        value: "openebs-hostpath"
      # LeaseTime defines the renewal period(in seconds) for client state
      #- name: LeaseTime
      #  value: 30
      # GraceTime defines the recovery period(in seconds) to reclaim locks
      #- name: GraceTime
      #  value: 30
      # FilePermissions defines the file ownership and mode specifications
      # for the NFS server's shared filesystem volume.
      # File permission changes are applied recursively if the root of the
      # volume's filesystem does not match the specified value.
      # Volume-specific file permission configuration can be specified by
      # using the FilePermissions config key in the PVC YAML, instead of
      # the StorageClass's.
      #- name: FilePermissions
      #  data:
      #    UID: "1000"
      #    GID: "2000"
      #    mode: "0755"
provisioner: openebs.io/nfsrwx
reclaimPolicy: Delete
```

Above storageclass is using *openebs-hostpath* Storageclass as BackendStorageclass. You can change it to as required.

Once the Storageclass is successfully created, you can provision a volume by creating a PVC with the above storageclass. Sample PVC YAML is as below:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: "openebs-rwx"
  resources:
    requests:
      storage: 1Gi
```

To check the binding of PVC, run below command:
```
kubectl get pvc -n <PVC-NAMESPACE> <PVC-NAME>
```

Sample output is as below:
```
$ kubectl get pvc -n default nfs-pvc
NAME      STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
nfs-pvc   Bound    pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646   1Gi        RWX            openebs-rwx    14s
```
In above example, provisioner has created NFS PV named *pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646*.


NFS Provisioner creates required resources to back above PVC using name "nfs-<PV_NAME>". To check the list of resources created for this PVC,

- To check the Service:
```
$ kubectl get svc -n openebs nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646
NAME                                           TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)            AGE
nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646   ClusterIP   10.0.0.47    <none>        2049/TCP,111/TCP   1m
```

- To check the NFS Server deployment:
```
$ kubectl get deploy -n openebs nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646
NAME                                           READY   UP-TO-DATE   AVAILABLE   AGE
nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646   1/1     1            1           1m
```

- To check the Backend PVC:
```
$ kubectl get pvc -n openebs nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646
NAME                                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS       AGE
nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646   Bound    pvc-b2bf16db-e74c-4b50-8d45-412b511f250c   1Gi        RWO            openebs-hostpath   1m
```

- To check the Backend PV:
```
$ kubectl get pv pvc-b2bf16db-e74c-4b50-8d45-412b511f250c
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                                  STORAGECLASS       REASON   AGE
pvc-b2bf16db-e74c-4b50-8d45-412b511f250c   1Gi        RWO            Delete           Bound    openebs/nfs-pvc-b5d6caae-831c-4a4e-97d8-ddfe3ca9a646   openebs-hostpath            1m
```

## Delete NFS Volume
Since NFS PV is dynamically provisioned, you can delete NFS PV by deleting PVC.
To delete PVC created in [Provision NFS Volume](#provision-nfs-volume)

```
$ kubectl delete pvc nfs-pvc -n default
persistentvolumeclaim "nfs-pvc" deleted
```

If NFS PV is created with **reclaimPolicy: Retain**, you can delete the PV using below list of commands:
```
- kubectl delete deploy -n openebs nfs-<PV-NAME>
- kubectl delete svc -n openebs nfs-<PV-NAME>
- kubectl delete pvc -n openebs nfs-<PV-NAME>
```
Here *PV-NAME* is name of the NFS PV.


If you face any issue using NFS Provisioner, you can file an issue on github (https://github.com/openebs/dynamic-nfs-provisioner/issues), or talk to us on the [openebs slack](https://kubernetes.slack.com/messages/openebs/) community.
