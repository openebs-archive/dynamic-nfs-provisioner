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
		eventType     ProvisionerEventType
		obj           interface{}
		expectedObj   interface{}
		expectedError error
	}{
		{
			name: "when backendPVC hook is configured then PVC obj should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			resourceType:  ResourceBackendPVC,
			obj:           generateFakePvcObj("test", "pvc", nil, nil),
			expectedObj:   generateFakePvcObj("test", "pvc", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     ProvisionerEventCreate,
			expectedError: nil,
		},
		{
			name: "when backendPV hook is configured then PV obj should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
						Action:          HookActionAdd,
					},
				},
			},
			resourceType:  ResourceBackendPV,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     ProvisionerEventCreate,
			expectedError: nil,
		},
		{
			name: "when NFSService hook is configured then Service obj should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSService,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     ProvisionerEventCreate,
			expectedError: nil,
		},
		{
			name: "when NFSPV hook is configured then PV obj should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
						Action:      HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSPV,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     ProvisionerEventCreate,
			expectedError: nil,
		},
		{
			name: "when NFSDeployment hook is configured then Deployment obj should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
						Action:              HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeDeploymentObj("test", "deployment", nil, nil),
			expectedObj:   generateFakeDeploymentObj("test", "deployment", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			eventType:     ProvisionerEventCreate,
			expectedError: nil,
		},
		{
			name: "when backendPVC hook is configured and PV object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			resourceType:  ResourceBackendPVC,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", nil, nil),
			eventType:     ProvisionerEventCreate,
			expectedError: errors.Errorf("*v1.PersistentVolume is not a PersistentVolumeClaim type"),
		},
		{
			name: "when backendPV hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
						Action:          HookActionAdd,
					},
				},
			},
			resourceType:  ResourceBackendPV,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     ProvisionerEventCreate,
			expectedError: errors.Errorf("*v1.Service is not a PersistentVolume type"),
		},
		{
			name: "when NFSService hook is configured and PV object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSService,
			obj:           generateFakePvObj("pv", nil, nil),
			expectedObj:   generateFakePvObj("pv", nil, nil),
			eventType:     ProvisionerEventCreate,
			expectedError: errors.Errorf("*v1.PersistentVolume is not a Service type"),
		},
		{
			name: "when NFSPV hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
						Action:      HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSPV,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     ProvisionerEventCreate,
			expectedError: errors.Errorf("*v1.Service is not a PersistentVolume type"),
		},
		{
			name: "when NFSDeployment hook is configured and Service object is passed then object should not be modified and error should be returned",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
						Action:              HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeServiceObj("test", "service", nil, nil),
			expectedObj:   generateFakeServiceObj("test", "service", nil, nil),
			eventType:     ProvisionerEventCreate,
			expectedError: errors.Errorf("*v1.Service is not a Deployment type"),
		},
		{
			name: "when NFSDeployment hook is configured with different eventType, object shouldn't be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
						Action:              HookActionAdd,
					},
				},
			},
			resourceType:  ResourceNFSServerDeployment,
			obj:           generateFakeDeploymentObj("test", "deployment", nil, nil),
			expectedObj:   generateFakeDeploymentObj("test", "deployment", nil, nil),
			eventType:     ProvisionerEventDelete,
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
		eventType    ProvisionerEventType
		shouldExists bool
	}{
		// Backend PVC hook test
		{
			name: "when backendPVC Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventCreate,
			shouldExists: true,
		},
		{
			name: "when backendPVC Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventDelete,
			shouldExists: true,
		},
		{
			name: "when backendPVC Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is not added and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when backendPVC Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPVC,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},

		// Backend PV hook test
		{
			name: "when backendPV Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: true,
		},
		{
			name: "when backendPV Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: true,
		},
		{
			name: "when backendPV Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is not added and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when backendPV Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceBackendPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},

		// NFS Service hook test
		{
			name: "when NFSService Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventCreate,
			shouldExists: true,
		},
		{
			name: "when NFSService Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventDelete,
			shouldExists: true,
		},
		{
			name: "when NFSService Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is not added and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSService Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSService,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},

		// NFS PV hook test
		{
			name: "when NFSPV Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: true,
		},
		{
			name: "when NFSPV Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: true,
		},
		{
			name: "when NFSPV Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is not added and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSPV Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSPV,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},

		// NFS Deployment hook test
		{
			name: "when NFSServerDeployment Hook is added with event Create and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventCreate,
			shouldExists: true,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Delete and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventDelete,
			shouldExists: true,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Create and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is added with event Delete and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventDelete,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is not added and checking for event Create",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventCreate,
			shouldExists: false,
		},
		{
			name: "when NFSServerDeployment Hook is not added and checking for event Delete",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
					},
				},
			},
			resourceType: ResourceNFSServerDeployment,
			eventType:    ProvisionerEventDelete,
			shouldExists: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.hook, "hook should not be nil")
			assert.Equal(t, test.shouldExists, test.hook.ActionExists(test.resourceType, test.eventType))
		})
	}
}
