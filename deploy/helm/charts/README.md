#  OpenEBS NFS PV Provisioner

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Helm chart for openebs dynamic nfs provisioner. This chart bootstraps OpenEBS Dynamic NFS PV provisioner deployment on a [Kubernetes](http://kubernetes.io) cluster using the  [Helm](https://helm.sh) package manager.


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

Please visit the [link](https://openebs.github.io/dynamic-nfs-provisioner/) for install instructions via helm3.

```console
# Helm
$ helm install [RELEASE_NAME] openebs-nfs/nfs-provisioner
```

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._


## Uninstall Chart

```console
# Helm
$ helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
# Helm
$ helm upgrade [RELEASE_NAME] [CHART] --install
```


## Configuration

The following table lists the configurable parameters of the OpenEBS NFS PV Provisioner chart and their default values.

| Parameter                                   | Description                                   | Default                                   |
| ------------------------------------------- | --------------------------------------------- | ----------------------------------------- | 
| `analytics.enabled`                         | Enable sending stats to Google Analytics          | `true`                          |
| `imagePullSecrets`                          | Provides image pull secret                       | `""`                            |
| `nfsProvisioner.enabled`                             | Enable NFS PV Provisioner                          | `true`                          |
| `nfsProvisioner.image.registry`                      | Registry for NFS PV Provisioner image              | `""`                            |
| `nfsProvisioner.image.repository`                    | Image repository for NFS PV Provisioner            | `openebs/provisioner-nfs` |
| `nfsProvisioner.image.tag`                           | Image tag for NFS PV Provisioner	                  | `0.2.0`                         |
| `nfsProvisioner.image.pullPolicy`                    | Image pull policy for NFS PV Provisioner           | `IfNotPresent`                  |
| `nfsProvisioner.annotations`                         | Annotations for NFS PV Provisioner metadata        | `""`                            |
| `nfsProvisioner.nodeSelector`                        | Nodeselector for NFS PV Provisioner pods           | `""`                            |
| `nfsProvisioner.tolerations`                         | NFS PV Provisioner pod toleration values           | `""`                            |
| `nfsProvisioner.securityContext`                     | Security context for container                     | `""`                            |
| `nfsProvisioner.healthCheck.initialDelaySeconds`     | Delay before liveness probe is initiated          | `30`                            |
| `nfsProvisioner.healthCheck.periodSeconds`           | How often to perform the liveness probe           | `60`                            | 
| `nfsProvisioner.enableLeaderElection`                | Enable leader election                            | `true`                          |
| `rbac.create`                               | Enable RBAC Resources                             | `true`                          |
| `rbac.pspEnabled`                           | Create pod security policy resources              | `false`                         |
| `nfsProvisioner.affinity`                            | NFS PV Provisioner pod affinity                    | `{}`                            | 
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
helm install <release-name> -f values.yaml ----namespace openebs nfs-provisioner
```

> **Tip**: You can use the default [values.yaml](values.yaml)
