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

package provisioner

import (
	"fmt"
	"os"
	"testing"
	"time"

	errors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func getInt64Ptr(val int64) *int64 {
	return &val
}

func getFakePVCObject(pvcNamespace, pvcName, scName string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pvcNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
}

func getFakeDeploymentObject(namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getFakeServiceObject(namespace, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func verifyDeploymentExistence(namespace, name string) func(*appsv1.Deployment) error {
	return func(deployment *appsv1.Deployment) error {
		if deployment.Namespace != namespace {
			return errors.Errorf("expected deployment namespace %s but got %s", namespace, deployment.Namespace)
		}
		if deployment.Name != name {
			return errors.Errorf("expected deployment name %s but got %s", name, deployment.Name)
		}
		return nil
	}
}

func verifyDeploymentFSGIDValue(expectedFSGID *int64) func(*appsv1.Deployment) error {
	return func(deployment *appsv1.Deployment) error {
		fsGroup := deployment.Spec.Template.Spec.SecurityContext.FSGroup
		if fsGroup == nil && expectedFSGID != nil {
			return errors.Errorf("expected fsgroup to exist on deployment %s/%s but got nil", deployment.Namespace, deployment.Name)
		}

		if fsGroup != nil && expectedFSGID == nil {
			return errors.Errorf("expected fsgroup not to exist on deployment %s/%s but exist %d", deployment.Namespace, deployment.Name, *fsGroup)
		}

		if fsGroup != nil && expectedFSGID != nil && *fsGroup != *expectedFSGID {
			return errors.Errorf("expected deployment %s/%s to have fsGroup value %d but got %d", deployment.Namespace, deployment.Name, *expectedFSGID, *fsGroup)
		}
		return nil
	}
}

func verifyDeploymentEnvValues(envKey, envValue string) func(*appsv1.Deployment) error {
	return func(deployment *appsv1.Deployment) error {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "nfs-server" {
				var isENVMatched bool
				for _, env := range container.Env {
					if envKey == env.Name {
						isENVMatched = env.Value == envValue
						if !isENVMatched {
							return errors.Errorf("expected env %s to have %s but got %s", envKey, envValue, env.Value)
						}
						break
					}
				}
				if isENVMatched {
					return nil
				}
			}
		}
		return errors.Errorf("expected to have env key & value as %s:%s but env doesn't exist", envKey, envValue)
	}
}

func TestCreateBackendPVC(t *testing.T) {
	tests := map[string]struct {
		options           *KernelNFSServerOptions
		provisioner       *Provisioner
		preProvisionedPVC *corev1.PersistentVolumeClaim
		isErrExpected     bool
		expectedPVCName   string
	}{
		"when there are no errors PVC should get created": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test1-pv",
				capacity:            "5G",
				backendStorageClass: "test1-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			expectedPVCName: "nfs-test1-pv",
		},
		"when PVC is pre-provisioned": {
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test2-pv",
				capacity:            "5G",
				backendStorageClass: "test2-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
			expectedPVCName:   "nfs-test2-pv",
			preProvisionedPVC: getFakePVCObject("nfs-server-ns2", "nfs-test2-pv", "test2-sc"),
		},
		"when PVC is pre-provisioned with same name in provisioner namespace": {
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test3-pv",
				capacity:            "5G",
				backendStorageClass: "test3-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns3",
			},
			expectedPVCName:   "nfs-test3-pv",
			preProvisionedPVC: getFakePVCObject("openebs", "nfs-test3-pv", "test3-sc"),
		},
		"when provisioner is configured to mark resource for volume-events": {
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test4-pv",
				capacity:            "5G",
				backendStorageClass: "test4-sc",
			},
			provisioner: &Provisioner{
				kubeClient:                  fake.NewSimpleClientset(),
				serverNamespace:             "nfs-server-ns4",
				markResourceForVolumeEvents: true,
			},
			expectedPVCName:   "nfs-test4-pv",
			preProvisionedPVC: getFakePVCObject("openebs", "nfs-test3-pv", "test3-sc"),
		},
	}

	for name, test := range tests {
		if test.preProvisionedPVC != nil {
			_, err := test.provisioner.kubeClient.
				CoreV1().
				PersistentVolumeClaims(test.preProvisionedPVC.Namespace).
				Create(test.preProvisionedPVC)
			if err != nil {
				t.Errorf("failed to pre-create PVC %s/%s error: %v", test.preProvisionedPVC.Namespace, test.preProvisionedPVC.Name, err)
			}
		}

		err := test.provisioner.createBackendPVC(test.options)
		if test.isErrExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got %v", name, err)
		}

		if !test.isErrExpected {
			nfsPVCObj, err := test.provisioner.kubeClient.
				CoreV1().
				PersistentVolumeClaims(test.provisioner.serverNamespace).
				Get(test.expectedPVCName, metav1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get PVC %s/%s error: %v", test.provisioner.serverNamespace, test.expectedPVCName, err)
			} else {
				if test.expectedPVCName != nfsPVCObj.Name {
					t.Errorf("%q test failed expected PVC name %s but got %s", name, test.expectedPVCName, nfsPVCObj.Name)
				}

				if test.provisioner.markResourceForVolumeEvents {
					assert.True(t, eventFinalizerExists(&nfsPVCObj.ObjectMeta), "Finalizer for volume-event should be set")
					assert.False(t, eventAnnotationExists(&nfsPVCObj.ObjectMeta), "Annotation for volume-event should be set")
				} else {
					assert.False(t, eventFinalizerExists(&nfsPVCObj.ObjectMeta), "Finalizer for volume-event should be set")
					assert.False(t, eventAnnotationExists(&nfsPVCObj.ObjectMeta), "Annotation for volume-event should be set")
				}
			}
		}
	}
}

func TestDeleteBackendPVC(t *testing.T) {
	tests := map[string]struct {
		options       *KernelNFSServerOptions
		provisioner   *Provisioner
		existingPVC   *corev1.PersistentVolumeClaim
		isErrExpected bool
	}{
		"when there are no errors PVC should get deleted": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test1-pv",
				capacity:            "5G",
				backendStorageClass: "test1-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			existingPVC: getFakePVCObject("nfs-server-ns1", "nfs-test1-pv", "test1-sc"),
		},
		"when PVC is already deleted": {
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test2-pv",
				capacity:            "5G",
				backendStorageClass: "test2-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			if test.existingPVC != nil {
				_, err := test.provisioner.kubeClient.
					CoreV1().
					PersistentVolumeClaims(test.existingPVC.Namespace).
					Create(test.existingPVC)
				if err != nil {
					t.Errorf("failed to create existing PVC %s/%s error: %v", test.existingPVC.Namespace, test.existingPVC.Name, err)
				}
			}
			err := test.provisioner.deleteBackendPVC(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got error: %v", name, err)
			}

			// PVC shouldn't exist
			if !test.isErrExpected {
				_, err = test.provisioner.kubeClient.CoreV1().
					PersistentVolumeClaims(test.provisioner.serverNamespace).
					Get("nfs-"+test.options.pvName, metav1.GetOptions{})
				if !k8serrors.IsNotFound(err) {
					t.Errorf("%q test failed expected PVC %s/%s not to exist after deleting but got err: %v", name, test.provisioner.serverNamespace, test.options.pvName, err)
				}
			}
		})
	}
}

func TestCreateDeployment(t *testing.T) {
	tests := map[string]struct {
		options                  *KernelNFSServerOptions
		provisioner              *Provisioner
		preProvisionedDeployment *appsv1.Deployment
		isErrExpected            bool
		expectedDeploymentFields []func(*appsv1.Deployment) error
	}{
		"when there are no errors deployment should get created": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test1-pv",
				pvcName:       "nfs-test1-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			expectedDeploymentFields: []func(*appsv1.Deployment) error{
				verifyDeploymentExistence("nfs-server-ns1", "nfs-test1-pv"),
				verifyDeploymentFSGIDValue(nil),
				verifyDeploymentEnvValues("CUSTOM_EXPORTS_CONFIG", ""),
				verifyDeploymentEnvValues("NFS_LEASE_TIME", "0"),
				verifyDeploymentEnvValues("NFS_GRACE_TIME", "0"),
			},
		},
		"when deployment is pre-provisioned": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test2-pv",
				pvcName:       "nfs-test2-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
			expectedDeploymentFields: []func(*appsv1.Deployment) error{
				verifyDeploymentExistence("nfs-server-ns2", "nfs-test2-pv"),
			},
			preProvisionedDeployment: getFakeDeploymentObject("nfs-server-ns2", "nfs-test2-pv"),
		},
		"when deployment exist with same name in provisioner namespace": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test3-pv",
				pvcName:       "nfs-test3-pv",
				fsGroup:       getInt64Ptr(123),
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns3",
			},
			expectedDeploymentFields: []func(*appsv1.Deployment) error{
				verifyDeploymentExistence("nfs-server-ns3", "nfs-test3-pv"),
				verifyDeploymentFSGIDValue(getInt64Ptr(123)),
				verifyDeploymentEnvValues("CUSTOM_EXPORTS_CONFIG", ""),
				verifyDeploymentEnvValues("NFS_LEASE_TIME", "0"),
				verifyDeploymentEnvValues("NFS_GRACE_TIME", "0"),
			},
			preProvisionedDeployment: getFakeDeploymentObject("openebs", "nfs-test3-pv"),
		},
		"when NFS server options are specified then deployment should create with those": {
			options: &KernelNFSServerOptions{
				provisionerNS:         "openebs",
				pvName:                "test4-pv",
				pvcName:               "nfs-test4-pv",
				fsGroup:               getInt64Ptr(123),
				leaseTime:             100,
				graceTime:             100,
				nfsServerCustomConfig: "/nfsshare *(rw,fsid=0,async,no_auth_nlm)",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns4",
			},
			expectedDeploymentFields: []func(*appsv1.Deployment) error{
				verifyDeploymentExistence("nfs-server-ns4", "nfs-test4-pv"),
				verifyDeploymentFSGIDValue(getInt64Ptr(123)),
				verifyDeploymentEnvValues("CUSTOM_EXPORTS_CONFIG", "/nfsshare *(rw,fsid=0,async,no_auth_nlm)"),
				verifyDeploymentEnvValues("NFS_LEASE_TIME", "100"),
				verifyDeploymentEnvValues("NFS_GRACE_TIME", "100"),
			},
		},
	}
	os.Setenv(string(NFSServerImageKey), "openebs/nfs-server:ci")

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			if test.preProvisionedDeployment != nil {
				_, err := test.provisioner.kubeClient.
					AppsV1().
					Deployments(test.preProvisionedDeployment.Namespace).
					Create(test.preProvisionedDeployment)
				if err != nil {
					t.Errorf("failed to pre-create deployment %s/%s error: %v", test.preProvisionedDeployment.Namespace, test.preProvisionedDeployment.Name, err)
				}
			}

			err := test.provisioner.createDeployment(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got %v", name, err)
			}

			if !test.isErrExpected {
				deployName := "nfs-" + test.options.pvName
				nfsDeployObj, err := test.provisioner.kubeClient.
					AppsV1().
					Deployments(test.provisioner.serverNamespace).
					Get(deployName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get deployment %s/%s error: %v", test.provisioner.serverNamespace, deployName, err)
				} else {
					for _, fn := range test.expectedDeploymentFields {
						err = fn(nfsDeployObj)
						if err != nil {
							t.Errorf("%q test failed expected error not to occur but got %v", name, err)
						}
					}
				}
			}
		})
	}
	os.Unsetenv(string(NFSServerImageKey))
}

func TestDeleteDeployment(t *testing.T) {
	tests := map[string]struct {
		options            *KernelNFSServerOptions
		provisioner        *Provisioner
		existingDeployment *appsv1.Deployment
		isErrExpected      bool
	}{
		"when there are no errors deployment should get deleted": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test1-pv",
				pvcName:       "nfs-test1-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			existingDeployment: getFakeDeploymentObject("nfs-server-ns1", "nfs-test1-pv"),
		},
		"when deployment is already deleted": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test2-pv",
				pvcName:       "nfs-test2-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			if test.existingDeployment != nil {
				_, err := test.provisioner.kubeClient.
					AppsV1().
					Deployments(test.existingDeployment.Namespace).
					Create(test.existingDeployment)
				if err != nil {
					t.Errorf("failed to create existing deployment %s/%s error: %v", test.existingDeployment.Namespace, test.existingDeployment.Name, err)
				}
			}

			err := test.provisioner.deleteDeployment(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got %v", name, err)
			}

			if !test.isErrExpected {
				_, err := test.provisioner.kubeClient.
					AppsV1().
					Deployments(test.provisioner.serverNamespace).
					Get("nfs-"+test.options.pvName, metav1.GetOptions{})
				if !k8serrors.IsNotFound(err) {
					t.Errorf("%q test failed expected deployment not to exist but got error %v", name, err)
				}
			}
		})
	}
}

func TestCreateService(t *testing.T) {
	tests := map[string]struct {
		options               *KernelNFSServerOptions
		provisioner           *Provisioner
		preProvisionedService *corev1.Service
		isErrExpected         bool
		expectedServiceName   string
	}{
		"when there are no errors service should get created": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test1-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			expectedServiceName: "nfs-test1-pv",
		},
		"when service is pre-provisioned": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test2-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
			expectedServiceName:   "nfs-test2-pv",
			preProvisionedService: getFakeServiceObject("nfs-server-ns2", "nfs-test2-pv"),
		},
		"when service exist with same name in provisioner namespace": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test3-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns3",
			},
			expectedServiceName:   "nfs-test3-pv",
			preProvisionedService: getFakeServiceObject("openebs", "nfs-test3-pv"),
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			if test.preProvisionedService != nil {
				_, err := test.provisioner.kubeClient.CoreV1().
					Services(test.preProvisionedService.Namespace).
					Create(test.preProvisionedService)
				if err != nil {
					t.Errorf("failed to pre-create service %s/%s error: %v", test.preProvisionedService.Namespace, test.preProvisionedService.Name, err)
				}
			}

			err := test.provisioner.createService(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got %v", name, err)
			}

			if !test.isErrExpected {
				svcObj, err := test.provisioner.kubeClient.
					CoreV1().
					Services(test.provisioner.serverNamespace).
					Get(test.expectedServiceName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get service %s/%s error: %v", test.provisioner.serverNamespace, test.expectedServiceName, err)
				} else {
					if test.expectedServiceName != svcObj.Name {
						t.Errorf("%q test failed expected service name %s but got %s", name, test.expectedServiceName, svcObj.Name)
					}
				}
			}
		})
	}
}

func TestDeleteService(t *testing.T) {
	tests := map[string]struct {
		options         *KernelNFSServerOptions
		provisioner     *Provisioner
		existingService *corev1.Service
		isErrExpected   bool
	}{
		"when there are no errors service should get deleted": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test1-pv",
				pvcName:       "nfs-test1-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns1",
			},
			existingService: getFakeServiceObject("nfs-server-ns1", "nfs-test1-pv"),
		},
		"when service is already deleted": {
			options: &KernelNFSServerOptions{
				provisionerNS: "openebs",
				pvName:        "test2-pv",
				pvcName:       "nfs-test2-pv",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns2",
			},
		},
	}

	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			if test.existingService != nil {
				_, err := test.provisioner.kubeClient.
					CoreV1().
					Services(test.existingService.Namespace).
					Create(test.existingService)
				if err != nil {
					t.Errorf("failed to create existing deployment %s/%s error: %v", test.existingService.Namespace, test.existingService.Name, err)
				}
			}

			err := test.provisioner.deleteService(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got %v", name, err)
			}

			if !test.isErrExpected {
				_, err := test.provisioner.kubeClient.
					CoreV1().
					Services(test.provisioner.serverNamespace).
					Get("nfs-"+test.options.pvName, metav1.GetOptions{})
				if !k8serrors.IsNotFound(err) {
					t.Errorf("%q test failed expected service not to exist but got error %v", name, err)
				}
			}
		})
	}
}

func TestGetNFSServerAddress(t *testing.T) {
	tests := map[string]struct {
		options               *KernelNFSServerOptions
		provisioner           *Provisioner
		isErrExpected         bool
		expectedServiceIP     string
		shouldBoundBackendPvc bool
	}{
		"when there are no errors service address should be returned": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test1-pv",
				capacity:            "5G",
				backendStorageClass: "test1-sc",
			},
			provisioner: &Provisioner{
				kubeClient:        fake.NewSimpleClientset(),
				serverNamespace:   "nfs-server-ns1",
				backendPvcTimeout: 60 * time.Second,
			},
			expectedServiceIP:     "nfs-test1-pv.nfs-server-ns1.svc.cluster.local",
			shouldBoundBackendPvc: true,
		},
		"when opted for clusterIP it should service address": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test2-pv",
				capacity:            "5G",
				backendStorageClass: "test2-sc",
			},
			provisioner: &Provisioner{
				kubeClient:        fake.NewSimpleClientset(),
				serverNamespace:   "nfs-server-ns2",
				useClusterIP:      true,
				backendPvcTimeout: 60 * time.Second,
			},
			// Since we are using fake clients there won't be ClusterIP on service
			// so expecting for empty value
			expectedServiceIP:     "",
			shouldBoundBackendPvc: true,
		},
		"when backend PVC failed to bound": {
			// NOTE: Populated only fields required for test
			options: &KernelNFSServerOptions{
				provisionerNS:       "openebs",
				pvName:              "test3-pv",
				capacity:            "5G",
				backendStorageClass: "test3-sc",
			},
			provisioner: &Provisioner{
				kubeClient:      fake.NewSimpleClientset(),
				serverNamespace: "nfs-server-ns3",
				useClusterIP:    false,
			},
			// Since we are using fake clients there won't be ClusterIP on service
			// so expecting for empty value
			expectedServiceIP:     "",
			isErrExpected:         true,
			shouldBoundBackendPvc: false,
		},
	}
	os.Setenv(string(NFSServerImageKey), "openebs/nfs-server:ci")
	for name, test := range tests {
		name := name
		test := test
		informer := informers.NewSharedInformerFactory(test.provisioner.kubeClient, 0)
		pvcInformer := informer.Core().V1().PersistentVolumeClaims().Informer()
		pvcInformer.AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					if test.shouldBoundBackendPvc {
						boundPvc(test.provisioner.kubeClient, obj)
					}
				},
			},
		)
		stopCh := make(chan struct{})
		informer.Start(stopCh)
		assert.True(t, cache.WaitForCacheSync(stopCh, pvcInformer.HasSynced))

		t.Run(name, func(t *testing.T) {
			serviceIP, err := test.provisioner.getNFSServerAddress(test.options)
			if test.isErrExpected && err == nil {
				t.Errorf("%q test failed expected error to occur but got nil", name)
			}
			if !test.isErrExpected && err != nil {
				t.Errorf("%q test failed expected error not to occur but got %v", name, err)
			}

			if !test.isErrExpected {
				if test.expectedServiceIP != serviceIP {
					t.Errorf("%q test failed expected NFS Service address %q but got %q", name, test.expectedServiceIP, serviceIP)
				}
			}
		})
		close(stopCh)
	}
	os.Unsetenv(string(NFSServerImageKey))
}

func boundPvc(client kubernetes.Interface, obj interface{}) {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return
	}

	pvc.Status.Phase = corev1.ClaimBound

	_, err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(pvc)
	if err != nil {
		fmt.Printf("failed to update PVC object err=%+v\n", err)
	}
	return
}

func eventFinalizerExists(objMeta *metav1.ObjectMeta) bool {
	eventFinalizer := OpenebsEventFinalizerPrefix + OpenebsEventFinalizer

	for _, f := range objMeta.Finalizers {
		if f == eventFinalizer {
			return true
		}
	}
	return false
}

func eventAnnotationExists(objMeta *metav1.ObjectMeta) bool {
	for k, v := range objMeta.Annotations {
		if k == OpenebsEventAnnotation && v == "true" {
			return true
		}
	}
	return false
}
