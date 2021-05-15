# OpenEBS Dynamic NFS Provisioner Helm Repository

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

[Helm 3](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repo as follows:

```console
helm repo add openebs-nfs https://openebs.github.io/dynamic-nfs-provisioner
```

You can then run `helm search repo openebs-nfs` to see the charts.

#### Update OpenEBS Dynamic NFS Provisioner Repo

Once openebs-nfs repository has been successfully fetched into the local system, it has to be updated to get the latest version. The NFS Provisioner charts repo can be updated using the following command:

```console
helm repo update
```

#### Install using Helm 3

Run the following command to install the OpenEBS Dynamic NFS Provisioner helm chart using the default StorageClass as the Backend StorageClass:
```console
helm install [RELEASE_NAME] openebs-nfs/nfs-provisioner --namespace [NAMESPACE] --create-namespace
```

The chart requires a StorageClass to provision the backend volume for the NFS share. You can use the `--set-string nfsStorageClass.backendStorageClass=<storageclass-name>` flag in the `helm install` command to specify the Backend StorageClass. If a StorageClass is not specified, the default StorageClass is used.

Use the command below to get the name of the default StorageClasses in your cluster:

```console
kubectl get sc -o=jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{@.metadata.name}{"\n"}{end}'
```

Sample command to install the OpenEBS Dynamic NFS Provisioner helm chart using the default StorageClass as BackendStorageClass:

```console
helm install openebs-nfs openebs-nfs/nfs-provisioner --namespace openebs --create-namespace
```

If you do not have an available StorageClass, you can install the [OpenEBS Dynamic LocalPV Provisioner helm chart](https://openebs.github.io/dynamic-localpv-provisioner) and use the 'openebs-hostpath' StorageClass as Backend Storage Class. Sample commands:
```console
# Add openebs-localpv repo
helm repo add openebs-localpv https://openebs.github.io/dynamic-localpv-provisioner
helm repo update

# Install localpv-provisioner
helm install openebs-localpv openebs-localpv/localpv-provisioner -n openebs --create-namespace \
	--set openebsNDM.enabled=false \
	--set deviceClass.enabled=false \

# Install nfs-provisioner
helm install openebs-nfs openebs-nfs/nfs-provisioner -n openebs \
	--set-string nfsStorageClass.backendStorageClass="openebs-hostpath"
```

## Uninstall Chart

All NFS PVCs should be removed before installation.

```console
# Helm
helm uninstall [RELEASE_NAME] --namespace [NAMESPACE]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
# Helm
helm upgrade [RELEASE_NAME] [CHART] --install --namespace [NAMESPACE]
```


## Configuration

Refer to the OpenEBS Dynamic NFS Provisioner Helm chart [README.md file](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/deploy/helm/charts/README.md#configuration) for detailed configuration options.

