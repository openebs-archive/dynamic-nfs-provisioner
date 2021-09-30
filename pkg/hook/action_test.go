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

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildPVCHook(annotations map[string]string, finalizer []string) *PVCHook {
	return &PVCHook{
		Annotations: annotations,
		Finalizers:  finalizer,
	}
}

func buildPVHook(annotations map[string]string, finalizer []string) *PVHook {
	return &PVHook{
		Annotations: annotations,
		Finalizers:  finalizer,
	}
}

func buildServiceHook(annotations map[string]string, finalizer []string) *ServiceHook {
	return &ServiceHook{
		Annotations: annotations,
		Finalizers:  finalizer,
	}
}

func buildDeploymentHook(annotations map[string]string, finalizer []string) *DeploymentHook {
	return &DeploymentHook{
		Annotations: annotations,
		Finalizers:  finalizer,
	}
}

func generateFakePvcObj(ns, name string, annotations map[string]string, finalizers []string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Finalizers:  finalizers,
			Annotations: annotations,
		},
	}
}

func generateFakePvObj(name string, annotations map[string]string, finalizers []string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Finalizers:  finalizers,
			Annotations: annotations,
		},
	}
}

func generateFakeDeploymentObj(namespace, name string, annotations map[string]string, finalizers []string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Finalizers:  finalizers,
			Annotations: annotations,
		},
	}
}

func generateFakeServiceObj(namespace, name string, annotations map[string]string, finalizers []string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Finalizers:  finalizers,
			Annotations: annotations,
		},
	}
}

func TestAction(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		resourceType  int
		eventType     EventType
		obj           interface{}
		expectedObj   interface{}
		expectedError error
	}{
		{
			name: "when backendPVC hook is configured then PVC obj should be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceBackendPVC,
			obj:           generateFakePvcObj("test", "pvc", nil, nil),
			expectedObj:   generateFakePvcObj("test", "pvc", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when backendPV hook is configured then PV obj should be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceBackendPV,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when NFSService hook is configured then Service obj should be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceNFSService,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when NFSPV hook is configured then PV obj should be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceNFSPV,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when NFSDeployment hook is configured then Deployment obj should be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeDeploymentObj("test", "deployment", nil, nil),
			expectedObj:   generateFakeDeploymentObj("test", "deployment", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when backendPVC hook is configured with invalid actionEvent, object should not be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					"invalidActionEvent": HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceBackendPVC,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: nil,
		},
		{
			name: "when backendPVC hook is configured and PV object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceBackendPVC,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: errors.Errorf("*v1.PersistentVolume is not a PersistentVolumeClaim type"),
		},
		{
			name: "when backendPV hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceBackendPV,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: errors.Errorf("*v1.Service is not a PersistentVolume type"),
		},
		{
			name: "when NFSService hook is configured and PV object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceNFSService,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: errors.Errorf("*v1.PersistentVolume is not a Service type"),
		},
		{
			name: "when NFSPV hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
			},
			resourceType:  ResourceNFSPV,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: errors.Errorf("*v1.Service is not a PersistentVolume type"),
		},
		{
			name: "when NFSDeployment hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{

						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     EventTypeCreateVolume,
			expectedError: errors.Errorf("*v1.Service is not a Deployment type"),
		},
		{
			name: "when NFSDeployment hook is configured with different eventType, object shouldn't be modified",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeDeploymentObj("test", "deployment", nil, nil),
			expectedObj:   generateFakeDeploymentObj("test", "deployment", nil, nil),
			eventType:     EventTypeDeleteVolume,
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.hook, "hook should not be nil")
			err := test.hook.Action(test.obj, test.resourceType, test.eventType)
			if test.expectedError == nil {
				assert.Nil(t, err, "action should not return an error")
			} else {
				assert.Equal(t, err.Error(), test.expectedError.Error(), "error message should match")
			}
			assert.Equal(t, test.expectedObj, test.obj, "object should match")
		})
	}

}

func TestActionExists(t *testing.T) {
	tests := []struct {
		name         string
		hook         *Hook
		resourceType int
		eventType    EventType
		shouldExists bool
	}{
		// Backend PVC hook test
		{
			name: "when backendPVC Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeCreateVolume,
			shouldExists: true,
		},
		{
			name: "when backendPVC Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeDeleteVolume,
			shouldExists: true,
		},
		{
			name: "when backendPVC Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is not added and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPVC,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},

		// Backend PV hook test
		{
			name: "when backendPV Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: true,
		},
		{
			name: "when backendPV Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: true,
		},
		{
			name: "when backendPV Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is not added and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceBackendPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},

		// NFS Service hook test
		{
			name: "when NFSService Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeCreateVolume,
			shouldExists: true,
		},
		{
			name: "when NFSService Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeDeleteVolume,
			shouldExists: true,
		},
		{
			name: "when NFSService Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is not added and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSService,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},

		// NFS PV hook test
		{
			name: "when NFSPV Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						NFSPVConfig:     buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: true,
		},
		{
			name: "when NFSPV Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: true,
		},
		{
			name: "when NFSPV Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is not added and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSPV,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},

		// NFS Deployment hook test
		{
			name: "when NFSServerDeployment Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeCreateVolume,
			shouldExists: true,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeDeleteVolume,
			shouldExists: true,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnDeleteVolumeEvent: HookConfig{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is not added and checking for event Create",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeCreateVolume,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: map[ActionType]HookConfig{
					ActionAddOnCreateVolumeEvent: HookConfig{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
					},
				},
				Version: HookVersion,
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    EventTypeDeleteVolume,
			shouldExists: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.hook, "hook should not be nil")
			data, err := yaml.Marshal(test.hook)
			assert.Nil(t, err, "marshaling hook should not fail")
			hook, err := ParseHooks(data)
			assert.Nil(t, err, "parsing hook should not fail")
			assert.Equal(t, test.shouldExists, hook.ActionExists(test.resourceType, test.eventType))
		})
	}
}
