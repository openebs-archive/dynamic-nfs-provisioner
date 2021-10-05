# Dynamic NFS Volume Provisioner
[![Build Status](https://github.com/openebs/dynamic-nfs-provisioner/actions/workflows/build.yml/badge.svg)](https://github.com/openebs/dynamic-nfs-provisioner/actions/workflows/build.yml)
[![Go Report](https://goreportcard.com/badge/github.com/openebs/dynamic-nfs-provisioner)](https://goreportcard.com/report/github.com/openebs/dynamic-nfs-provisioner)
[![codecov](https://codecov.io/gh/openebs/dynamic-nfs-provisioner/branch/develop/graph/badge.svg)](https://app.codecov.io/gh/openebs/dynamic-nfs-provisioner)
[![Slack](https://img.shields.io/badge/chat!!!-slack-ff1493.svg?style=flat-square)](https://kubernetes.slack.com/messages/openebs)
[![BCH compliance](https://bettercodehub.com/edge/badge/openebs/dynamic-nfs-provisioner?branch=develop)](https://bettercodehub.com/results/openebs/dynamic-nfs-provisioner)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fopenebs%2Fdynamic-nfs-provisioner.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fopenebs%2Fdynamic-nfs-provisioner?ref=badge_shield)

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/HEAD/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

<p align="justify">
<strong>OpenEBS Dynamic NFS PV provisioner</strong> can be used to dynamically provision 
NFS Volumes using different kinds of block storage available on the Kubernetes nodes. 
<br>
<br>
</p>

## Project Status: Beta
Using NFS Volumes, you can share Volume data across the pods running on different node machines. You can easily create NFS Volumes using OpenEBS Dynamic NFS Provisioner and use it anywhere.

## Installing Dynamic NFS Provisioner
Please refer to our [Quickstart](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md) and the [OpenEBS Documentation](https://docs.openebs.io).


## Usage
[Deploying WordPress using Dynamic NFS Provisioner](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/workload/wordpress.md)

[Configuring Node Affinity for NFS Volumes](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/tutorial/node-affinity.md)

[Setting Resource requirements for NFS Server](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/tutorial/configure-nfs-server-resource-requirements.md)

[Configuring Hook for NFS Provisioner](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/tutorial/nfs-hook.md)

## Troubleshooting
If you encounter any issue while using OpenEBS Dynamic NFS Provisioner, review the [troubleshooting guide](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/troubleshooting.md). You can also [file an issue](https://github.com/openebs/dynamic-nfs-provisioner/issues) or talk to us on [#openebs channel](https://kubernetes.slack.com/messages/openebs) in the [Kubernetes Slack](https://kubernetes.slack.com).

## Contributing
OpenEBS welcomes your feedback and contributions in any form possible. To contribute code in OpenEBS Dynamic NFS Provisioner, please follow the instructions mentioned on [Contributing guide](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/CONTRIBUTING.md). If you need any help, you can chat with us on [#openebs-dev channel](https://kubernetes.slack.com/messages/openebs-dev) in the [Kubernetes Slack](https://kubernetes.slack.com).

## Roadmap
Find the Dynamic NFS Provisioner roadmap items at the [OpenEBS Roadmap page](https://github.com/orgs/openebs/projects/12).

### Code of conduct
Participation in the OpenEBS community is governed by the [CNCF Code of Conduct](CODE-OF-CONDUCT.md).

## Inspiration/Credit
- https://github.com/sjiveson/nfs-server-alpine

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fopenebs%2Fdynamic-nfs-provisioner.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fopenebs%2Fdynamic-nfs-provisioner?ref=badge_large)
