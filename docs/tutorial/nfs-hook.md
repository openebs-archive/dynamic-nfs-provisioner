# Using Hook in NFS Provisioner

OpenEBS Dynamic NFS Provisioner exposes Kubernetes PV using NFS server and enables it for ReadWriteMany(RWX) mode. To make this possible, NFS Provisioner creates few Kubernetes resources which are mapped to the generated NFS PV resources. Using hook, admin/user can enable the NFS Provisioner to add annotations and finalizers to NFS Resources. This tutorial explains, how to configure hook on NFS Provisioner.

If you haven't installed the NFS Provisioner, refer [QuickStart guide on How to install OpenEBS NFS Provisioner](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md#quickstart).

## Create Hook Configmap
First we need to create a Configmap resource in NFS Provisioner's namespace(i.e *openebs*).

Sample Configmap YAML is as below:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hook-config
  namespace: openebs
data:
  config: |
    hooks:
      addOrUpdateEntriesOnCreateVolumeEvent:
        backendPV:
          annotations:
            example.io/track: "true"
            example.io/owner: teamA
            example.io/tracking-create-time: $current-time
          finalizers:
          - example.io/tracking-protection
        backendPVC:
          annotations:
            example.io/track: "true"
            example.io/owner: teamA
          finalizers:
          - example.io/tracking-protection
        name: createHook
        nfsDeployment:
          annotations:
            example.io/track: "true"
            example.io/owner: teamA
          finalizers:
          - example.io/tracking-protection
        nfsPV:
          annotations:
            example.io/track: "true"
            example.io/owner: teamA
          finalizers:
          - example.io/tracking-protection
        nfsService:
          annotations:
            example.io/track: "true"
            example.io/owner: teamA
          finalizers:
          - example.io/tracking-protection
      removeEntriesOnDeleteVolumeEvent:
        backendPV:
          finalizers:
          - example.io/tracking-protection
        backendPVC:
          finalizers:
          - example.io/tracking-protection
        name: deleteHook
        nfsDeployment:
          finalizers:
          - example.io/tracking-protection
        nfsPV:
          finalizers:
          - example.io/tracking-protection
        nfsService:
          finalizers:
          - example.io/tracking-protection
    version: 1.0.0
```

Above hook config will add annotations and finalizers to NFS resources while creating NFS PV, and remove the finalizer from NFS resources while deleting NFS PV.

Hook configuration is having below structure:

```yaml
    hooks:
      actionWithEventType:
        backendPV:
          annotations:
          finalizers:
        backendPVC:
          annotations:
          finalizers:
        name: <HOOK_NAME>
        nfsDeployment:
          annotations:
          finalizers:
        nfsPV:
          annotations:
          finalizers:
        nfsService:
          annotations:
          finalizers:
    version: 1.0.0
```

Supported *actionWithEventType* are as below:
- addOrUpdateEntriesOnCreateVolumeEvent
    - This action will modify the resources, which are being created as part of the volume creation operation, by adding the provided configuration. If provided configuration exists in the resources spec then it will be updated with the given configuration.
- removeEntriesOnCreateVolumeEvent
    - This action will modify the resources, which are being created as part of the volume creation opreation, by removing the provided configuration from resource's spec. If provided configuration doesn't exists in the resource spec then it will skip those configuration.
- addOrUpdateEntriesOnDeleteVolumeEvent
    - This action will modify the resources of a NFS volume when it gets deleted, by adding the provided configuration. If provided configuration exists in the resource spec then it will be updated with the given config.
- removeEntriesOnDeleteVolumeEvent
    - This action will modify the resources of a NFS volume when it gets deleted, by removing the provided configuration from resource's spec. If provided configuration exists in the resource spec then it will be updated with the given config.

*Note:*
- *Duplicate **actionWithEventType** are not allowed.*
- *If hook is configured to add finalizers on NFS resources then you need to remove those finalizers through hook(by using **removeEntriesOnDeleteVolumeEvent**) or manually(to delete the NFS Volume).*

Above four *actionWithEventType* supports following resources:
- backendPV
    - supported fields
        - annotations
        - finalizers
- backendPVC
    - supported fields
        - annotations
        - finalizers
- nfsDeployment
    - supported fields
        - annotations
        - finalizers
- nfsService
    - supported fields
        - annotations
        - finalizers

## Updating NFS Provisioner
Once Hook Configmap is created, update the NFS Provisioner Deployment to use environment variable *OPENEBS_IO_NFS_HOOK_CONFIGMAP*.

To set the environment variable, run below command,
```bash
kubectl set env deployment/openebs-nfs-provisioner -n openebs   OPENEBS_IO_NFS_HOOK_CONFIGMAP=nfs-hook
```

Here, *nfs-hook* is the Configmap we created in [Create Hook Configmap](#create-hook-configmap)

After sucessful update, you can check the deployment spec to verify the environment name.

<details>
    <summary>Click to check sample yaml of nfs-provisioner with above env.</summary>

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "3"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"name":"openebs-nfs-provisioner","openebs.io/component-name":"openebs-nfs-provisioner","openebs.io/version":"dev"},"name":"openebs-nfs-provisioner","namespace":"openebs"},"spec":{"replicas":1,"selector":{"matchLabels":{"name":"openebs-nfs-provisioner","openebs.io/component-name":"openebs-nfs-provisioner"}},"strategy":{"type":"Recreate"},"template":{"metadata":{"labels":{"name":"openebs-nfs-provisioner","openebs.io/component-name":"openebs-nfs-provisioner","openebs.io/version":"dev"}},"spec":{"containers":[{"env":[{"name":"NODE_NAME","valueFrom":{"fieldRef":{"fieldPath":"spec.nodeName"}}},{"name":"OPENEBS_NAMESPACE","valueFrom":{"fieldRef":{"fieldPath":"metadata.namespace"}}},{"name":"OPENEBS_SERVICE_ACCOUNT","valueFrom":{"fieldRef":{"fieldPath":"spec.serviceAccountName"}}},{"name":"OPENEBS_IO_ENABLE_ANALYTICS","value":"false"},{"name":"OPENEBS_IO_NFS_SERVER_USE_CLUSTERIP","value":"true"},{"name":"OPENEBS_IO_INSTALLER_TYPE","value":"openebs-operator-nfs"},{"name":"OPENEBS_IO_NFS_SERVER_IMG","value":"openebs/nfs-server-alpine:ci"}],"image":"openebs/provisioner-nfs:ci","imagePullPolicy":"IfNotPresent","livenessProbe":{"exec":{"command":["sh","-c","test `pgrep \"^provisioner-nfs.*\"` = 1"]},"initialDelaySeconds":30,"periodSeconds":60},"name":"openebs-provisioner-nfs","resources":{"limits":{"cpu":"200m","memory":"200M"},"requests":{"cpu":"50m","memory":"50M"}}}],"serviceAccountName":"openebs-maya-operator"}}}}
  creationTimestamp: "2021-10-05T05:07:02Z"
  generation: 3
  labels:
    name: openebs-nfs-provisioner
    openebs.io/component-name: openebs-nfs-provisioner
    openebs.io/version: dev
  name: openebs-nfs-provisioner
  namespace: openebs
  resourceVersion: "19144"
  uid: b82a8a28-977f-4bfd-8240-a9e393fb4653
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: openebs-nfs-provisioner
      openebs.io/component-name: openebs-nfs-provisioner
  strategy:
    type: Recreate
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: openebs-nfs-provisioner
        openebs.io/component-name: openebs-nfs-provisioner
        openebs.io/version: dev
    spec:
      containers:
      - env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        - name: OPENEBS_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        - name: OPENEBS_SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.serviceAccountName
        - name: OPENEBS_IO_ENABLE_ANALYTICS
          value: "true"
        - name: OPENEBS_IO_NFS_SERVER_USE_CLUSTERIP
          value: "true"
        - name: OPENEBS_IO_INSTALLER_TYPE
          value: openebs-operator-nfs
        - name: OPENEBS_IO_NFS_SERVER_IMG
          value: openebs/nfs-server-alpine:ci
        - name: OPENEBS_IO_NFS_HOOK_CONFIGMAP
          value: hook-config
        image: openebs/provisioner-nfs:ci
        imagePullPolicy: IfNotPresent
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - test `pgrep "^provisioner-nfs.*"` = 1
          failureThreshold: 3
          initialDelaySeconds: 30
          periodSeconds: 60
          successThreshold: 1
          timeoutSeconds: 1
        name: openebs-provisioner-nfs
        resources:
          limits:
            cpu: 200m
            memory: 200M
          requests:
            cpu: 50m
            memory: 50M
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: openebs-maya-operator
      serviceAccountName: openebs-maya-operator
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2021-10-05T08:39:26Z"
    lastUpdateTime: "2021-10-05T08:39:26Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2021-10-05T05:07:02Z"
    lastUpdateTime: "2021-10-05T08:39:26Z"
    message: ReplicaSet "openebs-nfs-provisioner-5fd45558fc" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
```

</details>

## Creating NFS Volumes

Once NFS Provisioner is updated with environment variable *OPENEBS_IO_NFS_HOOK_CONFIGMAP*, you can start deploying NFS Volumes. NFS resources generated for the volumes will have the annotations and finalizers as mentioned in the hook Configmap.

If you need information on creating NFS Volume, visit [How to create NFS Volume](https://github.com/openebs/dynamic-nfs-provisioner/blob/develop/docs/intro.md#provision-nfs-volume)


