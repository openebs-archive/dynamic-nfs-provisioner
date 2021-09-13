# Deploy WordPress with NFS Persistent Volumes
This document explains how to deploy a WordPress site using OpenEBS NFS Volume. Since OpenEBS NFS volume supports RWX(ReadWriteMay) storage mode, WordPress deployment using OpenEBS NFS volumes is highly scalable.

## Prerequisites
Kubernetes cluster with OpenEBS NFS Provisioner installed. Refer [QuickStart guide on How to install OpenEBS NFS Provisioner](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md#quickstart)

## Deploying WordPress
We will use the Helm package to install WordPress in our kubernetes cluster. If you don't have Helm installed, follow the [Installing Helm](https://helm.sh/docs/intro/install/) guide for installation.

### Adding Help repo for WordPress
Use the below command to add helm repo for WordPress.

```
helm repo add bitnami https://charts.bitnami.com/bitnami
```

Once repo has been added successfully, update helm repo using the following command:

```
helm repo update
```

### Installing WordPress
Once the helm repo is added, you can install WordPress using `helm install` as mentioned below. In this command, we are using Storageclass `openebs-rwx` created using [QuickStart guide](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md#quickstart). If you have created a different OpenEBS NFS Storageclass then you need to update the value of `--set persistence.storageClass`.
You can also configure [the other parameters through the `--set` argument](https://github.com/bitnami/charts/tree/master/bitnami/wordpress#parameters)

```
helm install my-release -n wordpress --create-namespace \
       --set wordpressUsername=admin \
       --set wordpressPassword=password \
       --set mariadb.auth.rootPassword=secretpassword \
       --set persistence.storageClass=openebs-rwx \
       --set persistence.accessModes={ReadWriteMany} \
       --set volumePermissions.enabled=true \
       --set autoscaling.enabled=true \
       --set autoscaling.minReplicas=2 \
       --set autoscaling.maxReplicas=6 \
       --set autoscaling.targetCPU=80 \
        bitnami/wordpress
```

The above will create two WordPress application pods with RWX persistent volume. We are using `my-release` as a release name for the WordPress installation. You can replace `my-release` with a different name also.

You can check the generated pods using the command `kubectl get pods -n wordpress`.
```
$ kubectl get pods -n wordpress
NAME                                   READY   STATUS    RESTARTS   AGE
my-release-mariadb-0                   1/1     Running   0          3m14s
my-release-wordpress-79969f558-lqs56   1/1     Running   0          2m59s
my-release-wordpress-79969f558-qjblc   1/1     Running   0          3m14s
```

You can scale the WordPress deployment using a `kubectl scale` command as mentioned below:
```
$ kubectl scale --replicas=3 deployment/my-release-wordpress -n wordpress
deployment.apps/my-release-wordpress scaled
```

To check PVC/PV created for WordPress pods,
```
$ kubectl get pvc -n wordpress
NAME                        STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS       AGE
data-my-release-mariadb-0   Bound    pvc-9dfec460-fc8a-4033-b26c-a28637dcaa3e   8Gi        RWO            openebs-hostpath   3m33s
my-release-wordpress        Bound    pvc-0234ee9c-befc-4824-8230-3dd6779214cb   10Gi       RWX            openebs-rwx        3m33s
```

You can see PVC `my-release-wordpress` is using Storageclass `openebs-rwx` which is having `RWX` access mode.

## Clean up WordPress installation
To uninstall WordPress, run the below command:

```
helm uninstall my-release -n wordpress
```


To delete the `wordpress` namespace, run
```
kubectl delete ns wordpress
```
