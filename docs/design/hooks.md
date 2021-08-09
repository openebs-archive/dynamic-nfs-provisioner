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
- [Upgrade](#upgrade)

## Summary
This design is to implement hooks in nfs-provisioner to set annotation or finalizer on NFS PV resources during volume provisioning/deleting events. This document covers the definition of hooks and how to implement this.

## Goals
Set custom Annotation, Finalizer on nfs resources

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
NFS-Provisioner will use user-provided Configmap to learn hooks configuration. These hooks will be executed on volume provisioning or deleting events to set the given information on provided resources. 

### High-Level Design
User will deploy nfs-provisioner with Configmap having information about hook information. This Configmap should exist in the same namespace in which nfs-provisioner is deployed. Sample nfs-provisioner deployment config is as below:

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

User needs to provide Configmap name as a value of `OPENEBS_IO_HOOK_CONFIG` environment variable.

### Low-Level Design
#### Hook Definition
NFS Provisioner uses `Provisioner` to provision NFS PV. The existing `Provisioner` definition needs to be extended to use the following hooks definition.

```go
// ResourceType defines type of resource
type ResourceType string

const (
	// Type of resources created by nfs-provisioner
	ResourceBackendPVC          ResourceType = "BackendPVC"
	ResourceBackendPV           ResourceType = "BackendPV"
	ResourceNFSService          ResourceType = "NFSService"
	ResourceNFSPV               ResourceType = "NFSPV"
	ResourceNFSServerDeployment ResourceType = "NFSServerDeployment"

	// ResourceAll represent the all above resources
	ResourceAll ResourceType = "All"
)

// HookActionType defines type of action for annotation and finalizer
type HookActionType string

const (
	HookActionAdd    HookActionType = "Add"
	HookActionRemove HookActionType = "Remove"
)

// ProvisionerEventType defines the type of events on which hook needs to be executed
type ProvisionerEventType string

const (
	ProvisionerEventCreate ProvisionerEventType = "Create"
	ProvisionerEventDelete ProvisionerEventType = "Delete"
)

// HookResource represent the resources on which hook action
// needs to be executed
type HookResource struct {
	// List of Annotations, mapped to the type of resources, needs to be added on the given resource
	Annotations map[ResourceType][]string `json:"annotations"`

	// List of Finalizers, mapped to the type of resources, needs to be added on the given resource
	Finalizers map[ResourceType][]string `json:"finalizers"`
}

// HookConfig represent the to be executed by nfs-provisioner
type HookConfig struct {
	// Name represent hook name
	Name string `json:"name"`

	// Resource store the resources on which this hook will be executed
	Resource HookResource `json:"resource"`

	// Event defines provisioning event on which
	// given hook action needs to be executed
	Event ProvisionerEventType `json:"provisioningEvent"`

	// Action represent the type of hook action, i.e HookActionAdd or HookActionRemove
	Action HookActionType `json:"hookAction"`

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
    hooks []HookConfig
}
```

#### Configmap structure
User needs to create Configmap resource with name **hook-config** in nfs-provisioner namespace. Hook configuration needs to be provided in data field **config**.
Configmap needs be defined as below:

```yaml
apiVersion: v1
data:
  config: |
    - hookAction: Add
      name: hook1
      provisioningEvent: Create
      resource:
        annotations:
          BackendPV:
          - example.io/track=true
          - test.io/owner=teamA
          BackendPVC:
          - example.io/track=true
          - test.io/owner=teamA
          NFSPV:
          - example.io/track=true
          - test.io/owner=teamA
        finalizers:
          BackendPV:
          - example.io/tracking-protection
          - test.io/track=true
          BackendPVC:
          - example.io/tracking-protection
          - test.io/track=true
          NFSPV:
          - example.io/tracking-protection
          - test.io/track=true
      version: 1.0.0
    - hookAction: Remove
      name: hook2
      provisioningEvent: Delete
      resource:
        annotations:
          BackendPV:
          - example.io/track=true
          - test.io/owner=teamA
          BackendPVC:
          - example.io/track=true
          - test.io/owner=teamA
          NFSPV:
          - example.io/track=true
          - test.io/owner=teamA
        finalizers:
          BackendPV:
          - test.io/track=true
          BackendPVC:
          - test.io/track=true
          NFSPV:
          - test.io/track=true
      version: 1.0.0
kind: ConfigMap
metadata:
  name: hook-config
  namespace: openebs
```

#### NFS Provisioner changes
NFS Provisioner needs to lookup Configmap named **hook-config** in nfs-provisioner namespace. If Configmap exists then provisioner needs to initialize **Provisioner** with hook configuration provided in Configmap.
NFS Provisioner executes two events, volume provisioning, and volume deletion. On these two events provisioner needs to execute all the hooks as per the given hook Action.

## Upgrade
No specific action is required to upgrade the older version to use hooks.

HookConfig supports version tracking which includes major,minor and patch version. If any changes are added in future, this version number should be updated accordingly.
