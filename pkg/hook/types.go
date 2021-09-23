/*
Copyright 2021 The OpenEBS Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hook

// HookVersion represent the hook config version
const HookVersion = "1.0.0"

const (
	// Type of resources created by nfs-provisioner
	ResourceBackendPVC int = iota
	ResourceBackendPV
	ResourceNFSService
	ResourceNFSPV
	ResourceNFSServerDeployment
)

// ActionType defines type of action for the hook entry
type ActionType string

// ActionOp represent the operation performed for ActionType
type ActionOp string

const (
	// Note:
	// On adding new Action, ActionForEventMap must be updated with the new Action

	// ActionAddOnCreateVolumeEvent represent add action on volume create Event
	ActionAddOnCreateVolumeEvent ActionType = "addOrUpdateEntriesOnCreateVolumeEvent"

	// ActionRemoveOnCreateVolumeEvent represent remove action on volume create Event
	ActionRemoveOnCreateVolumeEvent ActionType = "removeEntriesOnCreateVolumeEvent"

	// ActionAddOnDeleteVolumeEvent represent add action on volume delete Event
	ActionAddOnDeleteVolumeEvent ActionType = "addOrUpdateEntriesOnDeleteVolumeEvent"

	// ActionRemoveOnDeleteVolumeEvent represent remove action on volume delete Event
	ActionRemoveOnDeleteVolumeEvent ActionType = "removeEntriesOnDeleteVolumeEvent"
)

const (
	// ActionOpAddOrUpdate define Action addOrUpdateEntries
	ActionOpAddOrUpdate ActionOp = "addOrUpdateEntries"

	// ActionOpRemove define Action removeEntries
	ActionOpRemove ActionOp = "removeEntries"
)

// EventType defines the type of events on which hook needs to be executed
type EventType string

const (
	// EventTypeCreateVolume represent volume create event
	EventTypeCreateVolume EventType = "CreateVolume"

	// EventTypeDeleteVolume represent volume delete event
	EventTypeDeleteVolume EventType = "DeleteVolume"
)

var (
	// ActionForEventMap stores the supported EventType for all ActionType
	ActionForEventMap = map[ActionType]struct {
		evType EventType
		actOp  ActionOp
	}{
		ActionAddOnCreateVolumeEvent:    {evType: EventTypeCreateVolume, actOp: ActionOpAddOrUpdate},
		ActionRemoveOnCreateVolumeEvent: {evType: EventTypeCreateVolume, actOp: ActionOpRemove},
		ActionAddOnDeleteVolumeEvent:    {evType: EventTypeDeleteVolume, actOp: ActionOpAddOrUpdate},
		ActionRemoveOnDeleteVolumeEvent: {evType: EventTypeDeleteVolume, actOp: ActionOpRemove},
	}
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
	Config map[ActionType]HookConfig `json:"hooks"`

	// Version represent HookConfig format version; includes major, minor and patch version
	Version string `json:"version"`

	// Following field is for internal use of hook

	// availableActions keep inventory of resources and events for which action is configured
	// in Hook.Config
	availableActions map[EventType]map[int]struct{}
}
