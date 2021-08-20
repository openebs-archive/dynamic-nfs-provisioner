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

const (
	// Type of resources created by nfs-provisioner
	ResourceBackendPVC int = iota
	ResourceBackendPV
	ResourceNFSService
	ResourceNFSPV
	ResourceNFSServerDeployment
)

// HookActionType defines type of action for annotation and finalizer
type HookActionType string

const (
	// HookActionAdd represent add action
	HookActionAdd HookActionType = "Add"
	// HookActionAdd represent remove action
	HookActionRemove HookActionType = "Remove"
)

// ProvisionerEventType defines the type of events on which hook needs to be executed
type ProvisionerEventType string

const (
	// ProvisionerEventCreate represent create event
	ProvisionerEventCreate ProvisionerEventType = "Create"
	// ProvisionerEventDelete represent delete event
	ProvisionerEventDelete ProvisionerEventType = "Delete"
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
	NFSPVConfig *PVHook `json:"NFSPV,omitempty"`

	// BackendPVConfig represent config for BackendPV resource
	BackendPVConfig *PVHook `json:"backendPV,omitempty"`

	// BackendPVCConfig represent config for BackendPVC resource
	BackendPVCConfig *PVCHook `json:"backendPVC,omitempty"`

	// NFSServiceConfig represent config for NFS Service resource
	NFSServiceConfig *ServiceHook `json:"NFSService,omitempty"`

	// NFSDeploymentConfig represent config for NFS Deployment resource
	NFSDeploymentConfig *DeploymentHook `json:"NFSDeployment,omitempty"`

	// Event defines provisioning event on which
	// given hook action needs to be executed
	Event ProvisionerEventType `json:"provisioningEvent"`

	// Action represent the type of hook action, i.e HookActionAdd or HookActionRemove
	Action HookActionType `json:"hookAction"`
}

// Hook stores HookConfig and its version
type Hook struct {
	//Config represent the list of HookConfig
	Config []HookConfig `json:"hooks"`

	// Version represent HookConfig format version; includes major, minor and patch version
	Version string `json:"version"`
}
