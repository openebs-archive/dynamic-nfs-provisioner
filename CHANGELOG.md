0.11.0 / 2023-12-08
========================
* add support for specifying mountOptions to default StorageClass ([#164](https://github.com/openebs/dynamic-nfs-provisioner/pull/164),[@pentago](https://github.com/farcaller) [@dsharma-dc](https://github.com/dsharma-dc))
* update documentation on backend storage prerequisite ([#165](https://github.com/openebs/dynamic-nfs-provisioner/pull/165),[@matthew-williams](https://github.com/matthew-williams) [@dsharma-dc](https://github.com/dsharma-dc))
* update google analytics to use GA4 ([#174](https://github.com/openebs/dynamic-nfs-provisioner/pull/174),[@Abhinandan-Purkait](https://github.com/Abhinandan-Purkait))
* add a toggle to allow/disallow garbage collection for the backend PVC ([#167](https://github.com/openebs/dynamic-nfs-provisioner/pull/167),[@njuptlzf](https://github.com/njuptlzf))

0.10.0 / 2023-02-09
========================
* add original PVC context to the nfs server deployment labels ([#151](https://github.com/openebs/dynamic-nfs-provisioner/pull/151),[@farcaller](https://github.com/farcaller))
* fix config generation logic -- prefer SC config over PVC config ([@niladrih](https://github.com/niladrih))

0.9.0 / 2022-01-03
========================
* support for changing shared filesystem ownership and mode ([#125](https://github.com/openebs/dynamic-nfs-provisioner/pull/125),[@niladrih](https://github.com/niladrih))

0.8.0 / 2021-11-09
========================
* send install analytic event for nfs-provisioner ([#109](https://github.com/openebs/dynamic-nfs-provisioner/pull/109),[@mynktl](https://github.com/mynktl))
* support to configure *ImagePullSecret* for nfs-server pods ([#114](https://github.com/openebs/dynamic-nfs-provisioner/pull/114),[@g-linville](https://github.com/g-linville))
* hooks support for nfs-provisioner, to add custom annotations/finalizers on nfs resources ([#93](https://github.com/openebs/dynamic-nfs-provisioner/pull/93),[@mynktl](https://github.com/mynktl))


0.7.1 / 2021-09-16
========================
* added installation analytic event for nfs-provisioner ([#110](https://github.com/openebs/dynamic-nfs-provisioner/pull/110),[@mynktl](https://github.com/mynktl))


0.7.0 / 2021-09-14
========================
* support to configure resource request and limit for NFS server resources ([#92](https://github.com/openebs/dynamic-nfs-provisioner/pull/92),[@mittachaitu](https://github.com/mittachaitu))
* added GHCR repository for docker images [provisioner-nfs](https://github.com/openebs/dynamic-nfs-provisioner/pkgs/container/provisioner-nfs) and [nfs-server-alpine](https://github.com/openebs/dynamic-nfs-provisioner/pkgs/container/nfs-server-alpine) ([#101](https://github.com/openebs/dynamic-nfs-provisioner/pull/101),[@niladrih](https://github.com/niladrih))


0.6.1 / 2021-08-30
========================
* support to configure resource request and limit for NFS server resources ([#92](https://github.com/openebs/dynamic-nfs-provisioner/pull/92),[@mittachaitu](https://github.com/mittachaitu))


0.6.0 / 2021-08-16
========================
* added garbage-collector to remove stale NFS resources ([#80](https://github.com/openebs/dynamic-nfs-provisioner/pull/80),[@mynktl](https://github.com/mynktl))
* support to configure timeout for Backed PVC binding ([#84](https://github.com/openebs/dynamic-nfs-provisioner/pull/84),[@mynktl](https://github.com/mynktl))


0.5.0 / 2021-07-16
========================
* adding metrics for nfs-provisioner ([#51](https://github.com/openebs/dynamic-nfs-provisioner/pull/51),[@mynktl](https://github.com/mynktl))
* support to access nfs-share volume by non-root application ([#52](https://github.com/openebs/dynamic-nfs-provisioner/pull/52),[@mittachaitu](https://github.com/mittachaitu))
* support to provision nfs resources in user provided namespace ([#58](https://github.com/openebs/dynamic-nfs-provisioner/pull/58),[@mynktl](https://github.com/mynktl))
* support to specify node affinity rules of NFS Server ([#59](https://github.com/openebs/dynamic-nfs-provisioner/pull/59),[@mittachaitu](https://github.com/mittachaitu))
