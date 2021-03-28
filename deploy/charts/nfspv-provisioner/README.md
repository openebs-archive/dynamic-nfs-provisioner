#  OpenEBS NFSPV Provisioner

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Helm chart for openebs dynamic nfspv provisioner. This chart bootstraps OpenEBS Dynamic NFSPV provisioner deployment on a [Kubernetes](http://kubernetes.io) cluster using the  [Helm](https://helm.sh) package manager.


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
helm repo add openebs-nfspv https://openebs.github.io/dynamic-nfspv-provisioner
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

Please visit the [link](https://openebs.github.io/dynamic-nfspv-provisioner/) for install instructions via helm3.

```console
# Helm
$ helm install [RELEASE_NAME] openebs-nfspv/nfspv-provisioner
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

The following table lists the configurable parameters of the OpenEBS LocalPV Provisioner chart and their default values.

| Parameter                                   | Description                                   | Default                                   |
| ------------------------------------------- | --------------------------------------------- | ----------------------------------------- | 
| `analytics.enabled`                         | Enable sending stats to Google Analytics          | `true`                          |
| `imagePullSecrets`                          | Provides image pull secrect                       | `""`                            |
| `nfspv.enabled`                             | Enable NFSPV Provisioner                          | `true`                          |
| `nfspv.image.registry`                      | Registry for LocalPV Provisioner image            | `""`                            |
| `nfspv.image.repository`                    | Image repository for NFSPV Provisioner            | `openebs/localpv-provisioner`   |
| `nfspvpv.image.tag`                         |	Image tag for NFSPV Provisioner	                  | `0.2.0`                         |
| `nfspv.image.pullPolicy`                    | Image pull policy for NFSPV Provisioner           | `IfNotPresent`                  |
| `nfspv.annotations`                         | Annotations for NFSPV Provisioner metadata        | `""`                            |
| `nfspv.nodeSelector`                        | Nodeselector for NFSPV Provisioner pods           | `""`                            |
| `nfspv.tolerations`                         | NFSPV Provisioner pod toleration values           | `""`                            |
| `nfspv.securityContext`                     | Seurity context for container                     | `""`                            |
| `nfspv.healthCheck.initialDelaySeconds`     | Delay before liveness probe is initiated          | `30`                            |
| `nfspv.healthCheck.periodSeconds`           | How often to perform the liveness probe           | `60`                            | 
| `nfspv.enableLeaderElection`                | Enable leader election                            | `true`                          |
| `rbac.create`                               | Enable RBAC Resources                             | `true`                          |
| `rbac.pspEnabled`                           | Create pod security policy resources              | `false`                         |
| `nfspv.affinity`                            | NFSPV Provisioner pod affinity                    | `{}`                            | 
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
helm install <release-name> -f values.yaml ----namespace openebs nfspv-provisioner
```

> **Tip**: You can use the default [values.yaml](values.yaml)
