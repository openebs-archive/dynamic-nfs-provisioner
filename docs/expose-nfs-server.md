# Exposing NFS Server

OpenEBS Dynamic NFS Provisioner provides NFS share volume by exposing kubernetes Persistent Volumes through NFS server. NFS Provisioner exposes NFS Server using Kubernetes Service resource and can be accessed inside the cluster using this Service. NFS Server can be accessed from outside the cluster also, by using ingress controller. This document explains the possible ways to expose NFS Server outside the cluster.

If you haven't installed the NFS Provisioner, refer [QuickStart guide on How to install OpenEBS NFS Provisioner](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md#quickstart).

## Table of contents
- [Exposing NFS Server using NodePort](#exposing-nfs-server-using-nodeport)
  - [Creating a PVC](#creating-a-pvc)
  - [Updating Service Type to NodePort](#updating-service-type-to-nodeport)
  - [Mounting NFS Volume](#mounting-nfs-volume)
- [Exposing NFS Server using Nginx Ingress](#exposing-nfs-server-using-nginx-ingress)
  - [Creating a PVC](#creating-a-pvc-ingress)
  - [Installing Nginx ingress controller](#installing-nginx-ingress-controller)
  - [Configuring Nginx ingress controller](#configuring-nginx-ingress-controller)
  - [Mounting NFS Volume](#mounting-nfs-volume-ingress)

## Exposing NFS Server using NodePort

This example list the steps to expose NFS Server using NodePort.

### Creating a PVC

First, We will create a NFS PV using below YAML.
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: openebs-rwx-pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: "openebs-rwx"
  resources:
    requests:
      storage: 1Gi
```

To check status for above PVC, run
```bash
kubectl get pvc  openebs-rwx-pvc
```

Above command will return the PVC information similar to below output:
```bash
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
openebs-rwx-pvc   Bound    pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27   1Gi        RWX            openebs-rwx    113s
```

### Updating Service Type to NodePort

NFS Provisioner exposes NFS Server using Service named *nfs-<NFS_PV_NAME>*.

To fetch the NFS Service information, run
```bash
kubectl get svc -n openebs  nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27 -o yaml
```

Above command will return the Service information similar to below output:
```yaml
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: "2021-09-20T12:35:40Z"
  name: nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27
  namespace: openebs
  resourceVersion: "19729"
  uid: 305e08be-d8df-4012-b003-4fbc7239bf2e
spec:
  clusterIP: 10.0.0.152
  clusterIPs:
  - 10.0.0.152
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: nfs
    port: 2049
    protocol: TCP
    targetPort: 2049
  - name: rpcbind
    port: 111
    protocol: TCP
    targetPort: 111
  selector:
    openebs.io/nfs-server: nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

Edit the above Service using `kubectl edit svc -n openebs nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27` and change *type*  from *ClusterIP* to *NodePort*.

Once NFS Service is updated, you can check the NodePort details using below command:
```bash
kubectl get svc  -n openebs  nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27
```

Above command will return the output similar to below output:
```bash
NAME                                           TYPE       CLUSTER-IP   EXTERNAL-IP   PORT(S)                        AGE
nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27   NodePort   10.0.0.152   <none>        2049:30994/TCP,111:32192/TCP   14m
```

From above output, node port *30994* is mapped to NFS port *2049*. We can mount the NFS Volume using port *30994*.

To get the external IP address detail, first lets find the node on which NFS Server pod is running.

To get the node name, run below command,
```bash
kubectl get pods -n openebs  -l openebs.io/nfs-server=nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27 -o wide
```
Here we are using label *openebs.io/nfs-server=nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27*, where value *nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27* is having *nfs-<NFS_PV_NAME>* format.

Above command should return the output similar to below output:
```bash
NAME                                                           READY   STATUS    RESTARTS   AGE   IP           NODE           NOMINATED NODE   READINESS GATES
nfs-pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27-996cf67f6-xtj4p   1/1     Running   0          26m   172.17.0.6   192.168.1.98   <none>           <none>
```

From above output, NFS Server is running on node *192.168.1.98*. To get the IP address of this node, run
```bash
kubectl get nodes  192.168.1.98 -o wide
```

Above command will return the output similar to below output:
```
NAME           STATUS   ROLES    AGE     VERSION         INTERNAL-IP    EXTERNAL-IP   OS-IMAGE                       KERNEL-VERSION    CONTAINER-RUNTIME
192.168.1.98   Ready    <none>   6h58m   v1.22.1-dirty   192.168.1.98   <none>        Debian GNU/Linux 10 (buster)   4.19.0-17-amd64   docker://18.9.1
```

From above output, IP address for node *192.168.1.98* is *192.168.1.98*. Now we can use this ip address and port *30994* to mount the NFS Volume.

### Mounting NFS Volume

To mount the NFS Volume *pvc-4ee1fd46-638d-47ba-a04d-af58137c3b27* outside the cluster, run
```bash
mount -t nfs  -o port=30994 192.168.1.98:/ nfs_mount
```

Above command will mount the NFS Volume at path `nfs_mount`.


## Exposing NFS Server using Nginx Ingress

This example list the steps to expose NFS Server using Ingress.

<h3 id="creating-a-pvc-ingress">
Creating a PVC
</h3>

First, We will create a NFS PV using below YAML.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: openebs-rwx-pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: "openebs-rwx"
  resources:
    requests:
      storage: 1Gi
```

To check status for above PVC, run
```bash
kubectl get pvc  openebs-rwx-pvc
```

Above command will return the PVC information similar to below output:
```bash
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
openebs-rwx-pvc   Bound    pvc-8b37730e-81f0-445c-91fe-72f1f04fda95   1Gi        RWX            openebs-rwx    6m12s
```

Verify NFS Service is created in *openebs* namespace, using below command:
```bash
$~ kubectl get svc  -n openebs  nfs-pvc-8b37730e-81f0-445c-91fe-72f1f04fda95
NAME                                           TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)            AGE
nfs-pvc-8b37730e-81f0-445c-91fe-72f1f04fda95   ClusterIP   10.245.175.162   <none>        2049/TCP,111/TCP   6m36s
```

### Installing Nginx ingress controller

If your custer doesn't have Nginx ingress controller then you can install the same using below command:
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.0.0/deploy/static/provider/cloud/deploy.yaml
```

*Please refer [Nginx Installation Guide](https://kubernetes.github.io/ingress-nginx/deploy/) for detailed information on Nginx controller.*

Above command will install the nginx controller in `ingress-nginx` namespace.

To check the status of nginx controller pod, run
```bash
kubectl get pods -n ingress-nginx
```

Above command will return the output similar to below:
```bash
NAME                                       READY   STATUS      RESTARTS   AGE
ingress-nginx-admission-create-tdfcx       0/1     Completed   0          7s
ingress-nginx-admission-patch-8clsz        0/1     Completed   0          7s
ingress-nginx-controller-fd7bb8d66-qq5hd   0/1     Running     0          7s
```

### Configuring Nginx ingress controller

To expose TCP Service using Nginx ingress, we need to enable Nginx controller to use configmap by adding `--tcp-services-configmap` pointing to an existing ConfigMap resource.
Before editing Nginx deployment, let's create a ConfigMap resource with NFS Service details as follow:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nfs-services
  namespace: ingress-nginx
data:
  20490: "openebs/nfs-pvc-8b37730e-81f0-445c-91fe-72f1f04fda95:2049"
```

In above ConfigMap, data should be in the format `PORT_NUM : "<NAMESPACE>/<SERVICE_NAME>:<SERVICE_PORT>"`

Once ConfigMap resource is created, update the Nginx deployment *ingress-nginx-controller* with following changes:
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: controller
          args:
            - /nginx-ingress-controller
            - --publish-service=$(POD_NAMESPACE)/ingress-nginx-controller
            - --election-id=ingress-controller-leader
            - --controller-class=k8s.io/ingress-nginx
            - --configmap=$(POD_NAMESPACE)/ingress-nginx-controller
            - --validating-webhook=:8443
            - --validating-webhook-certificate=/usr/local/certificates/cert
            - --validating-webhook-key=/usr/local/certificates/key
            - --tcp-services-configmap=$(POD_NAMESPACE)/nfs-services
          ports:
          - containerPort: 20490
            name: nfs
            protocol: TCP
```

Update the `ingress-nginx-controller` service to expose nginx port *20490* for nfs.
```yaml
apiVersion: v1
kind: Service
metadata:
  name: ingress-nginx-controller
  namespace: ingress-nginx
spec:
  ...
  ports:
  - name: nfs
    port: 20490
    protocol: TCP
    targetPort: 20490
  ...
  type: LoadBalancer
```

To fetch the external IP, run below command:
```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

Sample output for above command is as below:
```bash
NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)                                      AGE
ingress-nginx-controller   LoadBalancer   10.245.250.30   144.126.253.20   80:30353/TCP,20490:32177/TCP,443:30437/TCP   4m37s
```
Now we can mount NFS Volume using ip *144.126.253.20* and port *20490*.

<h3 id="mounting-nfs-volume-ingress">
Mounting NFS Volume
</h3>

To mount the NFS Volume *pvc-8b37730e-81f0-445c-91fe-72f1f04fda95* outside the cluster, run

```bash
mount -t nfs -o port=20490  144.126.253.20:/ nfs_mount
```

Above command will mount the NFS Volume at path `nfs_mount`.
