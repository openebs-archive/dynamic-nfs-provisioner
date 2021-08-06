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
    - [High Level Design](#high-level-design)
    - [Low Level Design](#low-level-design)
        - [Hook Definition](#hook-definition)
        - [Configmap structure](#configmap-structure)
        - [NFS Provisioner changes](#nfs-provisioner-changes)
- [Upgrade](#upgrade)

## Summary
This design is to implement hooks in nfs-provisioner to set annotation or finalizer on nfs pv resources during volume provisioning/deleting events. This document covers the definition of hooks and how to implement this.

## Goals
Set custom Annotation, Finalizer on nfs resources

## Proposal
### User Stories
#### Add custom annotation on NFS resources
I should be able to configure nfs-provisioner to set the provided annotation on resources created by nfs-provisioner.

#### Add custom finalizer on NFS resources
I should be able to configure nfs-provisioner to set the provided finalizer on resources created by nfs-provisioner. This finalizers should exists on resources if not marked to remove on volume deletion event.

#### Add custom annotation and finalizer on NFS resources
I should be able to configure nfs-provisioner to set the provided annotation and finalizer on resources created by nfs-provisioner.

### Proposed Implementation
NFS-Provisioner will use user-provided Configmap to learn hooks configuration. This hooks will be executed on volume provisioning or deleting events to set the given information on provided resources. 

### High Level Design
User will deploy nfs-provisioner with Configmap having information about hook information. This Configmap should exists in the same namespace in which nfs-provisioner is deployed. Sample nfs-provisioner deployment config is as below:

```yaml
    spec:
      serviceAccountName: openebs-maya-operator
      containers:
      - name: openebs-provisioner-nfs
        image: openebs/provisioner-nfs:ci
        env:
        - name: OPENEBS_IO_HOOK_CONFIG
          value: "nfs-hook"
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - test `pgrep "^provisioner-nfs.*"` = 1
          initialDelaySeconds: 30
          periodSeconds: 60
```

User need to provide Configmap name as value of `OPENEBS_IO_HOOK_CONFIG` environment variable.

### Low Level Design
#### Hook Definition
NFS Provisioner uses `Provisioner` to provision NFS PV. Existing `Provisioner` definition needs to be extended to use following hooks definition.

```go
// ResourceType defines type of resource
type ResourceType string

const (
	// Type of resources created by nfs-provisioner
	ResourceBackendPVC          ResourceType = "BackendPVC"
	ResourceBackendPV           ResourceType = "BackendPV"
	ResourceNFSService          ResourceType = "NfsService"
	ResourceNFSPV               ResourceType = "NfsPV"
	ResourceNFSServerDeployment ResourceType = "NfsServerDeployment"

	// ResourceAll represent the all above resources
	ResourceAll ResourceType = "All"
)

// HookActionType defines type of action for annotation and finalizer
type HookActionType string

const (
	HookActionAdd    HookActionType = "Add"
	HookActionRemove HookActionType = "Remove"
)

// ProvisionerEventType defines type of events on which hook needs to be executed
type ProvisionerEventType string

const (
	ProvisionerEventCreate ProvisionerEventType = "Create"
	ProvisionerEventDelete ProvisionerEventType = "Delete"
)

// HookConfig represent the hook to be executed by nfs-provisioner
type HookConfig struct {
	// Name represent hook name
	Name string `json:"name"`

	// List of Annotations, mapped to type of resources, needs to be added on given resource
	Annotations map[ResourceType][]string `json:"annotations"`

	// List of Finalizers, mapped to type of resources, needs to be added on given resource
	Finalizers map[ResourceType][]string `json:"finalizers"`

	// Event defines provisioning event on which
	// given hook action needs to be executed
	Event ProvisionerEventType `json:"provisioningEvent"`

	// Action represent the type of hook action, i.e HookActionAdd or HookActionRemove
	Action HookActionType `json:"hookAction"`
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
    hooks []HookConfig
}
```

#### Configmap structure
User need to create Configmap resource with name **hook-config** in nfs-provisioner namespace. Hook configuration needs to be provided in data field **config**.
Configmap needs be defined as below:

```yaml
apiVersion: v1
data:
  config: |
    - annotations:
        BackendPV:
        - example.io/track=true
        - test.io/owner=teamA
        BackendPVC:
        - example.io/track=true
        - test.io/owner=teamA
        NfsPV:
        - example.io/track=true
        - test.io/owner=teamA
      finalizers:
        BackendPV:
        - example.io/tracking-protection
        - test.io/track=true
        BackendPVC:
        - example.io/tracking-protection
        - test.io/track=true
        NfsPV:
        - example.io/tracking-protection
        - test.io/track=true
      hookAction: Add
      name: hook1
      provisioningEvent: Create
    - annotations:
        BackendPV:
        - example.io/track=true
        - test.io/owner=teamA
        BackendPVC:
        - example.io/track=true
        - test.io/owner=teamA
        NfsPV:
        - example.io/track=true
        - test.io/owner=teamA
      finalizers:
        BackendPV:
        - test.io/track=true
        BackendPVC:
        - test.io/track=true
        NfsPV:
        - test.io/track=true
      hookAction: Remove
      name: hook2
      provisioningEvent: Delete
kind: ConfigMap
metadata:
  name: hook-config
  namespace: openebs
```

#### NFS Provisioner changes
NFS Provisioner needs to lookup Configmap named **hook-config** in nfs-provisioner namespace. If Configmap exists then provisioner need to initialize **Provisioner** with hook configuration provided in Configmap.
NFS Provisioner executes two events, volume provisioning and volume deletion. On this two events provisioner needs to execute all the hook as per the given hook Action.

## Upgrade
No specific action required to upgrade older version to use hooks.

