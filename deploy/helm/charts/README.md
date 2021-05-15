#  OpenEBS NFS Provisioner

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Helm chart for openebs dynamic nfs provisioner. This chart bootstraps OpenEBS Dynamic NFS Provisioner deployment on a [Kubernetes](http://kubernetes.io) cluster using the  [Helm](https://helm.sh) package manager.


**Homepage:** <http://www.openebs.io/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| akhilerm | akhil.mohan@mayadata.io |  |
| kiranmova | kiran.mova@mayadata.io |  |
| prateekpandey14 | prateek.pandey@mayadata.io |  |
| rahulkrishnanra | rahulkrishnanfs@gmail.com |  |


## Get Repo Info

```console
helm repo add openebs-nfs https://openebs.github.io/dynamic-nfs-provisioner
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

The chart requires a StorageClass to provision the backend volume for the NFS share. If a StorageClass is not specified (use `--set-string nfsStorageClass.backendStorageClass` to specify -- details given below), the default StorageClass is used.

Please visit this [link](https://helm.sh/docs/) for helm 3 installation instructions.

```console
# Helm
helm install [RELEASE_NAME] openebs-nfs/nfs-provisioner --namespace [NAMESPACE] --create-namespace
```

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._


## Uninstall Chart

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

The following table lists the configurable parameters of the OpenEBS Dynamic NFS Provisioner chart and their default values. You can modify different parameters by specifying the desired value in the `helm install` command by using the `--set` and/or the `--set-string` flag(s).

In the following sample command we modify `nfsStorageClass.backendStorageClass` to specify the StorageClass to be used to provision the backend volume used for the NFS share. We also use `nfsStorageClass.isDefaultClass` to set an annotation such that the 'openebs-kernel-nfs' StorageClass is used as the default StorageClass for the cluster.

```console
helm install openebs-nfs openebs-nfs/nfs-provisioner --namespace openebs --create-namespace \
	--set-string nfsStorageClass.backendStorageClass="openebs-hostpath" \
	--set nfsStorageClass.isDefaultClass=true
```

| Parameter                             | Description                                   | Default                         |
| ------------------------------------- | --------------------------------------------- | ------------------------------- | 
| `analytics.enabled`                   | Enable sending stats to Google Analytics      | `true`                          |
| `fullnameOverride`                    | Set custom Full Name for resources. Defaults to ( Release-name + `nfsProvisioner.name` ) | `""`                            |
| `imagePullSecrets`                    | Provides image pull secret                    | `""`                            |
| `nameOverride`                        | Set custom name for resources. Defaults to `nfsProvisioner.name` | `""`                           |
| `nfsProvisioner.affinity`             | NFS Provisioner pod affinity                  | `{}`                            |
| `nfsProvisioner.enabled`              | Enable NFS Provisioner                        | `true`                          |
| `nfsProvisioner.enableLeaderElection` | Enable leader election                        | `true`                          |
| `nfsProvisioner.healthCheck.initialDelaySeconds` | Delay before liveness probe is initiated      | `30`                            |
| `nfsProvisioner.healthCheck.periodSeconds` | How often to perform the liveness probe        | `60`                           | 
| `nfsProvisioner.image.registry`       | Registry for NFS Provisioner image            | `""`                            |
| `nfsProvisioner.image.repository`     | Image repository for NFS Provisioner          | `openebs/provisioner-nfs`       |
| `nfsProvisioner.image.tag`            | Image tag for NFS Provisioner	                | `""`                            |
| `nfsProvisioner.image.pullPolicy`     | Image pull policy for NFS Provisioner image   | `IfNotPresent`                  |
| `nfsProvisioner.annotations`          | Annotations for NFS Provisioner metadata      | `""`                            |
| `nfsProvisioner.nodeSelector`         | Nodeselector for NFS Provisioner pod          | `""`                            |
| `nfsProvisioner.resources`            | Resource request and limit for the container  | `true`                          |
| `nfsProvisioner.securityContext`      | Security context for container                | `""`                            |
| `nfsProvisioner.tolerations`          | NFS Provisioner pod toleration values         | `""`                            |
| `nfsStorageClass.backendStorageClass` | StorageClass to be used to provision the backend volume. If not specified, the default StorageClass is used. | `""`                         |
| `nfsStorageClass.isDefaultClass`      | Make 'openebs-kernel-nfs' the default StorageClass | `"false"`                         |
| `nfsStorageClass.reclaimPolicy`       | ReclaimPolicy for NFS PVs                      | `"Delete"`                     |
| `rbac.create`                         | Enable RBAC Resources                          | `true`                         |
| `rbac.pspEnabled`                     | Create pod security policy resources           | `false`                        |
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
helm install <release-name> -f values.yaml ----namespace openebs openebs-nfs/nfs-provisioner --create-namespace
```

> **Tip**: You can use the default [values.yaml](values.yaml)
