# Upgrade

NFS Provisioner upgrade requires upgrading NFS Provisioner deployment and NFS server Deployment. It is not necessary to upgrade NFS server Deployment, unless mentioned in changelog/release-notes.

## Upgrade NFS Provisioner
### Installed using Helm
Before executing the helm upgrade, you need to download the latest chart. To update the helm repo with latest chart, run below command:

```bash
helm repo update
```

To upgrade nfs-provisioner to latest version, run below command:

```bash
helm upgrade nfs  openebs-nfs/nfs-provisioner -n openebs
```

Above command will update the nfs-provisioner to latest version. If you want to upgrade to a specific version, run below command:

```bash
helm upgrade nfs  openebs-nfs/nfs-provisioner -n openebs  --version=<DESIRED_VERSION>
```

*Note: In above command, `nfs` is helm repo name.*

### Installed using kubectl
If you have installed the nfs-provisioner through kubectl, then you can upgrade the nfs-provisioner deployment to latest version by running the below command:

```bash
kubectl apply -f https://openebs.github.io/charts/nfs-operator.yaml
```

Above command will upgrade the nfs-provisioner to latest version. You can also upgrade to specific version by running the below command:

```bash
kubectl apply -f https://openebs.github.io/charts/versioned/<OPENEBS VERSION>/nfs-operator.yaml
```

## Upgrading NFS server Deployment
To update the nfs-server deployment, run below command:

```bash
./docs/tutorial/upgrade-nfs-server.sh 0.7.1
```

Above command assumes that nfs-server deployments are running in *openebs* namespace. If you have configured nfs provisioner to create nfs-server deployment in different namespace, run below command:

```bash
./docs/tutorial/upgrade-nfs-server.sh -n <NFS_SERVER_NS> 0.7.1
```

*Note: Upgrading NFS server deployment recreates the nfs-server pod with the updated image tag. This action will cause downtime(**downtime = time to kill existing nfs-server pod + pull time for new nfs-server image + boot time for new nfs-server pod**) for IOs.*
