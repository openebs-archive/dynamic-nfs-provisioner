# Hooks to set Annotation, Finalizer on NFS resources

## Table of Contents
- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Goals](#goals)
- [Proposal](#proposal)
    - [User Stories](#user-stories)
        - [Add custom annotation on NFS resources](#add-custom-annotation-on-nfs-resources)
        - [Add custom finalizer on NFS resources](#add-custom-finalizer-on-nfs-resources)
        - [Add custom annotation and finalizer on NFS resources](#add-custom-annotation-and-finalizer-on-nfs-resources)
    - [Proposed Implementation](#proposed-implementation)
    - [High-Level Design](#high-level-design)
    - [Low-Level Design](#low-level-design)
        - [Hook Definition](#hook-definition)
        - [Configmap structure](#configmap-structure)
        - [NFS Provisioner changes](#nfs-provisioner-changes)
        - [Extending Hook](#extending-hook)
- [Upgrade](#upgrade)

## Summary
This design is to implement hooks in nfs-provisioner to set annotation or finalizer on NFS PV resources during volume provisioning/deleting events. This document covers the definition of hooks and how to implement this.

## Goals
Set custom Annotation, Finalizer on NFS resources

## Proposal
### User Stories
#### Add custom annotation on NFS resources
I should be able to configure nfs-provisioner to set the provided annotation on resources created by nfs-provisioner.

#### Add custom finalizer on NFS resources
I should be able to configure nfs-provisioner to set the provided finalizer on resources created by nfs-provisioner. These Finalizers should exist on resources if not marked to remove on volume deletion event.

#### Add custom annotation and finalizer on NFS resources
I should be able to configure nfs-provisioner to set the provided annotation and finalizer on resources created by nfs-provisioner.

#### Add custom annotation and finalizer on specific NFS resources
I should be able to configure nfs-provisioner to set the provided annotation and finalizer on specific resources created by nfs-provisioner.

### Proposed Implementation
NFS-Provisioner will use user-provided Config file to learn hooks configuration. These hooks will be executed on volume provisioning or deleting events to set the given information on provided resources. 

### High-Level Design
NFS Provisioner will load the hook configuration from the file located at pre-defined path. User can create a configmap with hook configuration and mount it at pre-defined path. Sample nfs-provisioner deployment config is as below:

```yaml
    spec:
      serviceAccountName: openebs-maya-operator
      containers:
      - name: openebs-provisioner-nfs
        image: openebs/provisioner-nfs:ci
        env:
        - name: OPENEBS_IO_NFS_HOOK_CONFIGMAP
          value: "nfs-hook"
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - test `pgrep "^provisioner-nfs.*"` = 1
          initialDelaySeconds: 30
          periodSeconds: 60
        volumeMounts:
          - mountPath: /etc/nfs-provisioner-hook
            name: hook-config
      volumes:
        - name: hook-config
          configMap:
            name: hook-config
```

User needs to create a configmap named 'hook-config' in provisioner's namespace.

### Low-Level Design
#### Hook Definition
NFS Provisioner uses `Provisioner` to provision NFS PV. The existing `Provisioner` definition needs to be extended to use the following hooks definition.

```go
// ActionType defines type of action for the hook entry
type ActionType string

const (
	// ActionAddOnCreateVolumeEvent represent add action on volume create Event
	ActionAddOnCreateVolumeEvent ActionType = "addOrUpdateEntriesOnCreateVolumeEvent"

	// ActionRemoveOnCreateVolumeEvent represent remove action on volume create Event
	ActionRemoveOnCreateVolumeEvent ActionType = "removeEntriesOnCreateVolumeEvent"

	// ActionAddOnDeleteVolumeEvent represent add action on volume delete Event
	ActionAddOnDeleteVolumeEvent ActionType = "addOrUpdateEntriesOnDeleteVolumeEvent"

	// ActionRemoveOnDeleteVolumeEvent represent remove action on volume delete Event
	ActionRemoveOnDeleteVolumeEvent ActionType = "removeEntriesOnDeleteVolumeEvent"
)

// PVHook defines the field which will be updated for PV Hook Action
type PVHook struct {
	// Annotations needs to be added/removed on/from the PV
	Annotations map[string]string `json:"annotations,omitempty"`

	// Finalizers needs to be added/removed on/from the PV
	Finalizers []string `json:"finalizers,omitempty"`
}

// PVCHook defines the field which will be updated for PVC Hook Action
type PVCHook struct {
	// Annotations needs to be added/removed on/from the PVC
	Annotations map[string]string `json:"annotations,omitempty"`

	// Finalizers needs to be added/removed on/from the PVC
	Finalizers []string `json:"finalizers,omitempty"`
}

// ServiceHook defines the field which will be updated for Service Hook Action
type ServiceHook struct {
	// Annotations needs to be added/removed on/from the Service
	Annotations map[string]string `json:"annotations,omitempty"`

	// Finalizers needs to be added/removed on/from the Service
	Finalizers []string `json:"finalizers,omitempty"`
}

// DeploymentHook defines the field which will be updated for Deployment Hook Action
type DeploymentHook struct {
	// Annotations needs to be added/removed on/from the Deployment
	Annotations map[string]string `json:"annotations,omitempty"`

	// Finalizers needs to be added/removed on/from the Deployment
	Finalizers []string `json:"finalizers,omitempty"`
}

// HookConfig represent the to be executed by nfs-provisioner
type HookConfig struct {
	// Name represent hook name
	Name string `json:"name"`

	// NFSPVConfig represent config for NFSPV resource
	NFSPVConfig *PVHook `json:"nfsPV,omitempty"`

	// BackendPVConfig represent config for BackendPV resource
	BackendPVConfig *PVHook `json:"backendPV,omitempty"`

	// BackendPVCConfig represent config for BackendPVC resource
	BackendPVCConfig *PVCHook `json:"backendPVC,omitempty"`

	// NFSServiceConfig represent config for NFS Service resource
	NFSServiceConfig *ServiceHook `json:"nfsService,omitempty"`

	// NFSDeploymentConfig represent config for NFS Deployment resource
	NFSDeploymentConfig *DeploymentHook `json:"nfsDeployment,omitempty"`
}

// Hook stores HookConfig and its version
type Hook struct {
	//Config represent the list of HookConfig
	Config [ActionType]HookConfig `json:"hooks"`

	// Version represent HookConfig format version; includes major, minor and patch version
	Version string `json:"version"`
}

//Provisioner struct has the configuration and utilities required
// across the different work-flows.
type Provisioner struct {
	stopCh chan struct{}

	kubeClient clientset.Interface

	// namespace in which provisioner is running
	namespace string

	// serverNamespace in which nfs server deployments gets created
	// can be set through env variable NFS_SERVER_NAMESPACE
	// default value is Provisioner.namespace
	serverNamespace string

	// defaultConfig is the default configurations
	// provided from ENV or Code
	defaultConfig []mconfig.Config

	// getVolumeConfig is a reference to a function
	getVolumeConfig GetVolumeConfigFn

	//determine if clusterIP or clusterDNS should be used
	useClusterIP bool

	// k8sNodeLister hold cache information about nodes
	k8sNodeLister listerv1.NodeLister

	// nodeAffinity specifies requirements for scheduling NFS Server
	nodeAffinity NodeAffinity

	/* New Field */
	// hooks which needs to be executed on provisioning events
	// Note: nfshook -> github.com/openebs/dynamic-nfs-provisioner/pkg/hook
	hook *nfshook.Hook
}
```

#### Configmap structure
User needs to create hook Configmap resource in nfs-provisioner namespace. Hook configuration needs to be provided in data field **config**.
Configmap needs be defined as below:

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
            test.io/owner: teamA
          finalizers:
          - test.io/tracking-protection
        backendPVC:
          annotations:
            example.io/track: "true"
            test.io/owner: teamA
          finalizers:
          - test.io/tracking-protection
        name: createHook
        nfsDeployment:
          annotations:
            example.io/track: "true"
            test.io/owner: teamA
          finalizers:
          - test.io/tracking-protection
        nfsPV:
          annotations:
            example.io/track: "true"
            test.io/owner: teamA
          finalizers:
          - test.io/tracking-protection
        nfsService:
          annotations:
            example.io/track: "true"
            test.io/owner: teamA
          finalizers:
          - test.io/tracking-protection
      removeEntriesOnDeleteVolumeEvent:
        backendPV:
          finalizers:
          - test.io/tracking-protection
        backendPVC:
          finalizers:
          - test.io/tracking-protection
        name: deleteHook
        nfsDeployment:
          finalizers:
          - test.io/tracking-protection
        nfsPV:
          finalizers:
          - test.io/tracking-protection
        nfsService:
          finalizers:
          - test.io/tracking-protection
    version: 1.0.0
```

User can also configure annotation or label value using template variable.
Below is snippet from Hook configuration with template variable.

```yaml
      addOrUpdateEntriesOnCreateVolumeEvent:
        backendPV:
          annotations:
            example.io/track: "true"
            test.io/owner: teamA
            test.io/tracking-create-time: $current-time
```
For initial development, following template variables will be supported:
```go
- $current-time // use current timestamp as value
```


#### NFS Provisioner changes
NFS Provisioner will initialize the hook configuration using pre-defined hook config file(*/etc/nfs-provisioner-hook/config*). If hook config file is not found then NFS Provisioner will skip the initialization of hooks and continue with provisioning.

NFS Provisioner executes two events, volume provisioning, and volume deletion. On these two events provisioner needs to execute all the hooks as per the given hook Action.

NFS Provisioner will initialize the hook on startup. If Hook configuration is invalid then NFS Provisioner will throw the error and exit.

While executing the hook, NFS Provisioner will execute the valid action only. If any invalid value or action is mentioned in the Hook then Provisioner will skip that specific entry.

Provisioner will verify the hook version mentioned in configmap with the provisioner's hook version. If the hook version is supported by the provisioner then it will process the hook (if the upgrade is required then it will update the configmap with an upgraded version). If hook version is not supported by the provisioner then it will not initialize the hook and return the error which will kill the pod eventually.

#### Extending Hook
As of now, This document covers design to add/remove only Annotation and Finalizer of the NFS resources. If required, Hook for relevant resource can be extended to modify the other field also. Since hook takes configuration in YAML format, new field definition should be added according to kubernetes definition only.

For example, To update **ImagePullSecrets** field of *NFSDeployment*, we can extend **DeploymentHook** as below:
```go
import (
  corev1 "k8s.io/api/core/v1"
)

// DeploymentHook defines the field which will be updated for Deployment Hook Action
type DeploymentHook struct {
	// Annotations needs to be added/removed on/from the Deployment
	Annotations map[string]string `json:"annotations,omitempty"`

	// Finalizers needs to be added/removed on/from the Deployment
	Finalizers []string `json:"finalizers,omitempty"`

	/* New Field */
	//ImagePullSecrets needs to be added/remove on/from the deployment
	ImagePullSecrets []corev1.LocalObjectReference
}
```


## Upgrade
No specific action is required to upgrade the older version to use hooks.

HookConfig supports version tracking which includes major,minor and patch version. If any changes are added in future, this version number should be updated accordingly.

Initial hook development will be versioned 1.0.0.
