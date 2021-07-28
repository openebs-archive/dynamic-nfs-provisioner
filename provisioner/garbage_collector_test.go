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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func generateFakePvcObj(ns, name, uid string, phase corev1.PersistentVolumeClaimPhase, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			UID:       types.UID(uid),
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: phase,
		},
	}
}

func generateFakePvObj(name string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec:   corev1.PersistentVolumeSpec{},
		Status: corev1.PersistentVolumeStatus{},
	}
}

func generateBackendPvcLabel(nfsPvcNs, nfsPvcName, nfsPvcUID, nfsPvName string) map[string]string {
	return map[string]string{
		nfsPvcNameLabelKey:    nfsPvcName,
		nfsPvcUIDLabelKey:     nfsPvcUID,
		nfsPvcNsLabelKey:      nfsPvcNs,
		"persistent-volume":   nfsPvName,
		"openebs.io/cas-type": "nfs-kernel",
	}
}

func getProvisioningTracker(pvName ...string) ProvisioningTracker {
	tracker := NewProvisioningTracker()

	for _, v := range pvName {
		tracker.Add(v)
	}

	return tracker
}

func TestRunGarbageCollector(t *testing.T) {
	GarbageCollectorInterval = 10 * time.Second

	nfsServerNs := "nfs-ns"

	clientset := fake.NewSimpleClientset()
	pvTracker := getProvisioningTracker()

	backendPvc := generateFakePvcObj(nfsServerNs, "nfs-pv5", "backend-pvc5-uid", corev1.ClaimBound,
		generateBackendPvcLabel("ns5", "pvc5", "uid5", "pv5"))
	nfsDeployment := getFakeDeploymentObject(nfsServerNs, "nfs-pv5")
	nfsService := getFakeServiceObject(nfsServerNs, "nfs-pv5")

	assert.NoError(t, createPvc(clientset, backendPvc), "on creating backend PVC resource")
	assert.NoError(t, createDeployment(clientset, nfsDeployment), "on creating nfs-server deployment resource")
	assert.NoError(t, createService(clientset, nfsService), "on creating nfs-server service resourec")

	stopCh := make(chan struct{})
	go RunGarbageCollector(clientset, pvTracker, nfsServerNs, stopCh)

	time.Sleep(GarbageCollectorInterval + 10*time.Second /* to ensure cleanUpStalePvc run */)
	close(stopCh)

	exists, err := pvcExists(clientset, backendPvc.Namespace, backendPvc.Name)
	assert.NoError(t, err, "checking backend PVC existence")
	assert.Equal(t, false, exists, "backend PVC %s hould be removed")

	exists, err = deploymentExists(clientset, nfsDeployment.Namespace, nfsDeployment.Name)
	assert.NoError(t, err, "checking nfs-server deployment existence")
	assert.Equal(t, false, exists, "nfs-server deployment should be removed")

	exists, err = serviceExists(clientset, nfsService.Namespace, nfsService.Name)
	assert.NoError(t, err, "checking nfs-server service existence")
	assert.Equal(t, false, exists, "nfs-server service should be removed")
}

func TestCleanUpStalePvc(t *testing.T) {
	nfsServerNs := "nfs-ns"

	tests := []struct {
		// name describe the test
		name string

		clientset *fake.Clientset
		pvTracker ProvisioningTracker

		nfsPvc        *corev1.PersistentVolumeClaim
		nfsPv         *corev1.PersistentVolume
		backendPvc    *corev1.PersistentVolumeClaim
		nfsDeployment *appsv1.Deployment
		nfsService    *corev1.Service

		shouldCleanup bool
	}{
		{
			name: "when NFS PVC is in bound state, NFS resources should not be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: false,

			nfsPvc: generateFakePvcObj("ns1", "pvc1", "uid1", corev1.ClaimBound, nil),
			nfsPv:  generateFakePvObj("pv1"),
			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv1", "backend-pvc1-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns1", "pvc1", "uid1", "pv1")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv1"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv1"),
		},
		{
			name: "when NFS PVC is in pending state, NFS resources should not be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: false,

			nfsPvc: generateFakePvcObj("ns2", "pvc2", "uid2", corev1.ClaimPending, nil),
			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv2", "backend-pvc2-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns2", "pvc2", "uid2", "pv2")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv2"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv2"),
		},
		{
			name: "when NFS PVC doesn't exist but provisioner is re-attempting provisioning, NFS resources should not be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker("pv3"),
			shouldCleanup: false,

			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv3", "backend-pvc3-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns3", "pvc3", "uid3", "pv3")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv3"),
		},
		{
			name: "when NFS PVC doesn't exist and NFS PV exists, NFS resources should not be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: false,

			nfsPv: generateFakePvObj("pv4"),
			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv4", "backend-pvc4-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns4", "pvc4", "uid1", "pv4")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv4"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv4"),
		},
		{
			name: "when NFS PVC and NFS PV doesn't exist, NFS resources should be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: true,

			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv5", "backend-pvc5-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns5", "pvc5", "uid5", "pv5")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv5"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv5"),
		},
		{
			name: "when PVC is having different UID and NFS PV exists, NFS resources should not be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: false,

			nfsPvc: generateFakePvcObj("ns6", "pvc6", "different-uid", corev1.ClaimBound, nil),
			nfsPv:  generateFakePvObj("pv6"),
			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv6", "backend-pvc6-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns6", "pvc6", "uid6", "pv6")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv6"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv6"),
		},
		{
			name: "when PVC is having different UID and NFS PV doesn't exist, NFS resources should be destroyed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: true,

			nfsPvc: generateFakePvcObj("ns7", "pvc7", "different-uid", corev1.ClaimBound, nil),
			backendPvc: generateFakePvcObj(nfsServerNs, "nfs-pv7", "backend-pvc7-uid", corev1.ClaimBound,
				generateBackendPvcLabel("ns7", "pvc7", "uid7", "pv7")),
			nfsDeployment: getFakeDeploymentObject(nfsServerNs, "nfs-pv7"),
			nfsService:    getFakeServiceObject(nfsServerNs, "nfs-pv7"),
		},
		{
			name: "when backend PVC is not having nfs-pvc labels, backend PVC should not be removed",

			clientset:     fake.NewSimpleClientset(),
			pvTracker:     getProvisioningTracker(),
			shouldCleanup: false,

			backendPvc: generateFakePvcObj(nfsServerNs, "not-nfs-pvc", "backend-pvc8-uid", corev1.ClaimPending,
				map[string]string{"openebs.io/cas-type": "nfs-kernel"}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, createPvc(test.clientset, test.nfsPvc), "on creating nfs PVC resource")
			assert.NoError(t, createPv(test.clientset, test.nfsPv), "on creating nfs PV resource")
			assert.NoError(t, createPvc(test.clientset, test.backendPvc), "on creating backend PVC resource")
			fmt.Printf("%+v\n", test.backendPvc)
			assert.NoError(t, createDeployment(test.clientset, test.nfsDeployment), "on creating nfs-server deployment resource")
			assert.NoError(t, createService(test.clientset, test.nfsService), "on creating nfs-server service resourec")

			assert.NoError(t, cleanUpStalePvc(test.clientset, test.pvTracker, nfsServerNs))

			if test.backendPvc != nil {
				exists, err := pvcExists(test.clientset, test.backendPvc.Namespace, test.backendPvc.Name)
				assert.NoError(t, err, "checking backend PVC existence")
				assert.NotEqual(t, test.shouldCleanup, exists, "backend PVC %s", ternary(test.shouldCleanup, "should be removed", "shouldn't be removed"))
			}

			if test.nfsDeployment != nil {
				exists, err := deploymentExists(test.clientset, test.nfsDeployment.Namespace, test.nfsDeployment.Name)
				assert.NoError(t, err, "checking nfs-server deployment existence")
				assert.NotEqual(t, test.shouldCleanup, exists, "nfs-server deployment %s", ternary(test.shouldCleanup, "should be removed", "shouldn't be removed"))
			}

			if test.nfsService != nil {
				exists, err := serviceExists(test.clientset, test.nfsService.Namespace, test.nfsService.Name)
				assert.NoError(t, err, "checking nfs-server service existence")
				assert.NotEqual(t, test.shouldCleanup, exists, "nfs-server service %s", ternary(test.shouldCleanup, "should be removed", "shouldn't be removed"))
			}

		})
	}

}

func ternary(cond bool, varA, varB interface{}) interface{} {
	if cond {
		return varA
	}
	return varB
}

func pvcExists(client *fake.Clientset, pvcNamespace, pvcName string) (bool, error) {
	_, err := client.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(pvcName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func deploymentExists(client *fake.Clientset, deploymentNs, deploymentName string) (bool, error) {
	_, err := client.AppsV1().Deployments(deploymentNs).Get(deploymentName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func serviceExists(client *fake.Clientset, serviceNs, serviceName string) (bool, error) {
	_, err := client.CoreV1().Services(serviceNs).Get(serviceName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

// createPvc creates PVC resource for the given PVC object.
// On successful creation or if object is nil, it return nil error,
// else return error, occured on create k8s resource
func createPvc(client *fake.Clientset, pvcObj *corev1.PersistentVolumeClaim) error {
	if pvcObj == nil {
		return nil
	}

	_, err := client.CoreV1().PersistentVolumeClaims(pvcObj.Namespace).Create(pvcObj)
	return err
}

// createDeployment creates Deployment resource for the given object
// on successful creation or if object is nil, it return nil error,
// else return error, occured on create k8s resource
func createDeployment(client *fake.Clientset, deployObj *appsv1.Deployment) error {
	if deployObj == nil {
		return nil
	}

	_, err := client.AppsV1().Deployments(deployObj.Namespace).Create(deployObj)
	return err
}

// createService creates Service resource for the given object
// on successful creation or if object is nil, it return nil error,
// else return error, occured on create k8s resource
func createService(client *fake.Clientset, serviceObj *corev1.Service) error {
	if serviceObj == nil {
		return nil
	}

	_, err := client.CoreV1().Services(serviceObj.Namespace).Create(serviceObj)
	return err
}

// createPv creates PV resource for the given PV object.
// On successful creation or if object is nil, it return nil error,
// else return error, occured on create k8s resource
func createPv(client *fake.Clientset, pvObj *corev1.PersistentVolume) error {
	if pvObj == nil {
		return nil
	}

	_, err := client.CoreV1().PersistentVolumes().Create(pvObj)
	return err
}
