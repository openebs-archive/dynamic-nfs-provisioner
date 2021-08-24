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

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExecuteHookOnNFSPV(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		PVName        string
		obj           *corev1.PersistentVolume
		expectedObj   *corev1.PersistentVolume
		shouldErrored bool
	}{

		{
			name: "when NFSPV hook is configured to add metadata but PV doesn't exist",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
					},
				},
			},
			PVName:        "pv1",
			shouldErrored: true,
		},
		{
			name: "when NFSPV hook is configured to add metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
						Action:      HookActionAdd,
					},
				},
			},
			PVName:        "pv2",
			obj:           generateFakePvObj("pv2", nil, nil),
			expectedObj:   generateFakePvObj("pv2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			shouldErrored: false,
		},
		{
			name: "when NFSPV hook is configured to remove metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:       ProvisionerEventCreate,
						Action:      HookActionRemove,
					},
				},
			},
			PVName:        "pv3",
			obj:           generateFakePvObj("pv3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj:   generateFakePvObj("pv3", nil, nil),
			shouldErrored: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if test.obj != nil {
				_, err := clientset.CoreV1().PersistentVolumes().Create(test.obj)
				assert.Nil(t, err, "PV creation failed, err=%s", err)
			}

			err := test.hook.ExecuteHookOnNFSPV(clientset, test.PVName, ProvisionerEventCreate)
			if test.shouldErrored {
				assert.NotNil(t, err, "ExecuteHookOnNFSPV should return error")
			} else {
				assert.Nil(t, err, "ExecuteHookOnNFSPV should not return error, err=%s", err)
			}

			if test.expectedObj == nil {
				return
			}

			pvObj, err := clientset.CoreV1().PersistentVolumes().Get(test.PVName, metav1.GetOptions{})
			assert.Nil(t, err, "failed to get PV=%s", test.PVName)
			assert.Equal(t, test.expectedObj, pvObj, "PV object should match")
		})
	}
}

func TestExecuteHookOnBackendPVC(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		ns            string
		pvcName       string
		obj           *corev1.PersistentVolumeClaim
		expectedObj   *corev1.PersistentVolumeClaim
		shouldErrored bool
	}{

		{
			name: "when BackendPVC hook is configured to add metadata but PVC doesn't exist",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			ns:            "ns1",
			pvcName:       "pvc1",
			shouldErrored: true,
		},
		{
			name: "when BackendPVC hook is configured to add metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			ns:            "ns2",
			pvcName:       "pvc2",
			obj:           generateFakePvcObj("ns2", "pvc2", nil, nil),
			expectedObj:   generateFakePvcObj("ns2", "pvc2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			shouldErrored: false,
		},
		{
			name: "when BackendPVC hook is configured to remove metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVCConfig: buildPVCHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionRemove,
					},
				},
			},
			ns:            "ns3",
			pvcName:       "pvc3",
			obj:           generateFakePvcObj("ns3", "pvc3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj:   generateFakePvcObj("ns3", "pvc3", nil, nil),
			shouldErrored: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if test.obj != nil {
				_, err := clientset.CoreV1().PersistentVolumeClaims(test.ns).Create(test.obj)
				assert.Nil(t, err, "PVC creation failed, err=%s", err)
			}

			err := test.hook.ExecuteHookOnBackendPVC(clientset, test.ns, test.pvcName, ProvisionerEventCreate)
			if test.shouldErrored {
				assert.NotNil(t, err, "ExecuteHookOnBackendPVC should return error")
			} else {
				assert.Nil(t, err, "ExecuteHookOnBackendPVC should not return error, err=%s", err)
			}

			if test.expectedObj == nil {
				return
			}

			pvcObj, err := clientset.CoreV1().PersistentVolumeClaims(test.ns).Get(test.pvcName, metav1.GetOptions{})
			assert.Nil(t, err, "failed to get PVC=%s/%s", test.ns, test.pvcName)
			assert.Equal(t, test.expectedObj, pvcObj, "PVC object should match")
		})
	}
}

func TestExecuteHookOnNFSService(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		ns            string
		svcName       string
		obj           *corev1.Service
		expectedObj   *corev1.Service
		shouldErrored bool
	}{

		{
			name: "when Service hook is configured to add metadata but PVC doesn't exist",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
					},
				},
			},
			ns:            "ns1",
			svcName:       "svc1",
			shouldErrored: true,
		},
		{
			name: "when Service hook is configured to add metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionAdd,
					},
				},
			},
			ns:            "ns2",
			svcName:       "svc2",
			obj:           generateFakeServiceObj("ns2", "svc2", nil, nil),
			expectedObj:   generateFakeServiceObj("ns2", "svc2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			shouldErrored: false,
		},
		{
			name: "when Service hook is configured to remove metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSServiceConfig: buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:            ProvisionerEventCreate,
						Action:           HookActionRemove,
					},
				},
			},
			ns:            "ns3",
			svcName:       "svc3",
			obj:           generateFakeServiceObj("ns3", "svc3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj:   generateFakeServiceObj("ns3", "svc3", nil, nil),
			shouldErrored: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if test.obj != nil {
				_, err := clientset.CoreV1().Services(test.ns).Create(test.obj)
				assert.Nil(t, err, "Service creation failed, err=%s", err)
			}

			err := test.hook.ExecuteHookOnNFSService(clientset, test.ns, test.svcName, ProvisionerEventCreate)
			if test.shouldErrored {
				assert.NotNil(t, err, "ExecuteHookOnNFSService should return error")
			} else {
				assert.Nil(t, err, "ExecuteHookOnNFSService should not return error, err=%s", err)
			}

			if test.expectedObj == nil {
				return
			}

			svcObj, err := clientset.CoreV1().Services(test.ns).Get(test.svcName, metav1.GetOptions{})
			assert.Nil(t, err, "failed to get Service=%s/%s", test.ns, test.svcName)
			assert.Equal(t, test.expectedObj, svcObj, "Service object should match")
		})
	}
}

func TestExecuteHookOnNFSDeployment(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		ns            string
		deployName    string
		obj           *appsv1.Deployment
		expectedObj   *appsv1.Deployment
		shouldErrored bool
	}{

		{
			name: "when Deployment hook is configured to add metadata but PVC doesn't exist",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
					},
				},
			},
			ns:            "ns1",
			deployName:    "deploy1",
			shouldErrored: true,
		},
		{
			name: "when Deployment hook is configured to add metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
						Action:              HookActionAdd,
					},
				},
			},
			ns:            "ns2",
			deployName:    "deploy2",
			obj:           generateFakeDeploymentObj("ns2", "deploy2", nil, nil),
			expectedObj:   generateFakeDeploymentObj("ns2", "deploy2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			shouldErrored: false,
		},
		{
			name: "when Deployment hook is configured to remove metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						NFSDeploymentConfig: buildDeploymentHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:               ProvisionerEventCreate,
						Action:              HookActionRemove,
					},
				},
			},
			ns:            "ns3",
			deployName:    "deploy3",
			obj:           generateFakeDeploymentObj("ns3", "deploy3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj:   generateFakeDeploymentObj("ns3", "deploy3", nil, nil),
			shouldErrored: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if test.obj != nil {
				_, err := clientset.AppsV1().Deployments(test.ns).Create(test.obj)
				assert.Nil(t, err, "Deployment creation failed, err=%s", err)
			}

			err := test.hook.ExecuteHookOnNFSDeployment(clientset, test.ns, test.deployName, ProvisionerEventCreate)
			if test.shouldErrored {
				assert.NotNil(t, err, "ExecuteHookOnNFSDeployment should return error")
			} else {
				assert.Nil(t, err, "ExecuteHookOnNFSDeployment should not return error, err=%s", err)
			}

			if test.expectedObj == nil {
				return
			}

			deployObj, err := clientset.AppsV1().Deployments(test.ns).Get(test.deployName, metav1.GetOptions{})
			assert.Nil(t, err, "failed to get Deployment=%s/%s", test.ns, test.deployName)
			assert.Equal(t, test.expectedObj, deployObj, "Deployment object should match")
		})
	}
}

func buildBackendPVCObj(ns, name, boundedPV string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: boundedPV,
		},
	}
}

func TestExecuteHookOnBackendPV(t *testing.T) {
	tests := []struct {
		name          string
		hook          *Hook
		ns            string
		pvcName       string
		pvcObj        *corev1.PersistentVolumeClaim
		obj           *corev1.PersistentVolume
		expectedObj   *corev1.PersistentVolume
		shouldErrored bool
	}{

		{
			name: "when BackendPV hook is configured to add metadata but PVC doesn't exist",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
					},
				},
			},
			ns:            "ns1",
			pvcName:       "pvc1",
			shouldErrored: true,
		},
		{
			name: "when BackendPV hook is configured to add metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
						Action:          HookActionAdd,
					},
				},
			},
			ns:            "ns2",
			pvcName:       "pvc2",
			pvcObj:        buildBackendPVCObj("ns2", "pvc2", "pv2"),
			obj:           generateFakePvObj("pv2", nil, nil),
			expectedObj:   generateFakePvObj("pv2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			shouldErrored: false,
		},
		{
			name: "when BackendPV hook is configured to remove metadata, object should be modified",
			hook: &Hook{
				Config: []HookConfig{
					{
						BackendPVConfig: buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
						Event:           ProvisionerEventCreate,
						Action:          HookActionRemove,
					},
				},
			},
			ns:            "ns3",
			pvcName:       "pvc3",
			pvcObj:        buildBackendPVCObj("ns3", "pvc3", "pv3"),
			obj:           generateFakePvObj("pv3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj:   generateFakePvObj("pv3", nil, nil),
			shouldErrored: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			if test.pvcObj != nil {
				_, err := clientset.CoreV1().PersistentVolumeClaims(test.ns).Create(test.pvcObj)
				assert.Nil(t, err, "PVC creation failed, err=%s", err)
			}

			if test.obj != nil {
				_, err := clientset.CoreV1().PersistentVolumes().Create(test.obj)
				assert.Nil(t, err, "PV creation failed, err=%s", err)
			}

			err := test.hook.ExecuteHookOnBackendPV(clientset, test.ns, test.pvcName, ProvisionerEventCreate)
			if test.shouldErrored {
				assert.NotNil(t, err, "ExecuteHookOnBackendPV should return error")
			} else {
				assert.Nil(t, err, "ExecuteHookOnBackendPV should not return error, err=%s", err)
			}

			if test.expectedObj == nil {
				return
			}

			pvObj, err := clientset.CoreV1().PersistentVolumes().Get(test.obj.Name, metav1.GetOptions{})
			assert.Nil(t, err, "failed to get PV=%s", test.pvcName)
			assert.Equal(t, test.expectedObj, pvObj, "PV object should match")
		})
	}
}
