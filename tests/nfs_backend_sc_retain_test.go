/*
Copyright 2021 The OpenEBS Authors

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

package tests

import (
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
)

var _ = Describe("TEST BACKEND PV EXISTENCE WITH BACKEND SC HAVING RETAIN POLICY", func() {
	var (
		applicationNamespace = "default"

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "retain-pvc-backend-pv"

		// nfs provisioner values
		openebsNamespace       = "openebs"
		scName                 = "nfs-backend-sc-retain"
		backendScName          = "backend-sc-retain"
		backendScBindingMode   = storagev1.VolumeBindingWaitForFirstConsumer
		backendScReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		scNfsServerType        = "kernel"
		// backendPvcName stores backend pvc name created by nfs pvc
		backendPvcName = ""
		// backendPvName stores backend pv name created by nfs pvc
		backendPvName = ""
	)

	When(fmt.Sprintf("create backend storageclass %s with reclaimPolicy=%s", backendScName, backendScReclaimPolicy), func() {
		It("should create backed storageclass", func() {
			By("creating storageclass")
			casObj := []mayav1alpha1.Config{
				{
					Name:  "StorageType",
					Value: "hostpath",
				},
				{
					Name:  "BasePath",
					Value: "/tmp/openebs",
				},
			}

			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			err = Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: backendScName,
					Annotations: map[string]string{
						string(mayav1alpha1.CASTypeKey):   "local",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner:       "openebs.io/local",
				VolumeBindingMode: &backendScBindingMode,
				ReclaimPolicy:     &backendScReclaimPolicy,
			})
			Expect(err).To(BeNil(), "while creating SC{%s}", scName)
		})
	})

	When(fmt.Sprintf("create storageclass with backendStorageclass=%s", backendScName), func() {
		It("should create storageclass", func() {
			By("creating storageclass")
			casObj := []mayav1alpha1.Config{
				{
					Name:  provisioner.KeyPVNFSServerType,
					Value: scNfsServerType,
				},
				{
					Name:  provisioner.KeyPVBackendStorageClass,
					Value: backendScName,
				},
			}

			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			err = Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: scName,
					Annotations: map[string]string{
						string(mayav1alpha1.CASTypeKey):   "nfsrwx",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner: "openebs.io/nfsrwx",
			})
			Expect(err).To(BeNil(), "while creating SC{%s}", scName)
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is created", scName), func() {
		It("should create a pvc ", func() {
			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).To(BeNil(), "while building pvc %s/%s object", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)

			_, err = Client.waitForPVCBound(pvcName, applicationNamespace)
			Expect(err).To(BeNil(), "while waiting %s/%s pvc to bound", applicationNamespace, pvcName)

			pvcObj, err = Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			backendPvcName = "nfs-" + pvcObj.Spec.VolumeName
			_, err = Client.waitForPVCBound(backendPvcName, openebsNamespace)
			Expect(err).To(BeNil(), "while waiting %s/%s pvc to bound", openebsNamespace, backendPvcName)

			pvcObj, err = Client.getPVC(openebsNamespace, backendPvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			backendPvName = pvcObj.Spec.VolumeName
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			By(fmt.Sprintf("pvc with storageclass %s is deleted", scName))
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)

			maxRetryCount := 10
			isPvcDeleted := false
			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPVC(applicationNamespace, pvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					isPvcDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}
			Expect(isPvcDeleted).To(BeTrue(), "pvc should be deleted")

			isBackendPvcDeleted := false
			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPVC(openebsNamespace, backendPvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					isBackendPvcDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}
			Expect(isBackendPvcDeleted).To(BeTrue(), "backend pvc should be deleted")
		})
	})

	When("verify backend PV state after application PVC deletion", func() {
		It("should have backend PV in released state", func() {
			Expect(backendPvName).NotTo(BeNil(), "backend PV name should not be empty")

			backendPvObj, err := Client.getPV(backendPvName)
			Expect(err).To(BeNil(), "while fetching pv {%s}", backendPvName)
			Expect(backendPvObj.Status.Phase).To(Equal(corev1.VolumeReleased), "while verifying backed PV bound phase")
		})
	})

	When("cleaning up backend PV", func() {
		It("should delete backend PV", func() {
			Expect(backendPvName).NotTo(BeNil(), "backend PV name should not be empty")

			err = Client.deletePV(backendPvName)
			Expect(err).To(BeNil(), "while deleting pv {%s}", backendPvName)
		})
	})

	When(fmt.Sprintf("StorageClass %s is deleted", scName), func() {
		It("should delete the storageclass", func() {
			By("deleting storageclass")
			err := Client.deleteStorageClass(scName)
			Expect(err).To(BeNil(), "while deleting sc {%s}", scName)
		})
	})

	When(fmt.Sprintf("backend storageclass %s is deleted", backendScName), func() {
		It("should delete the storageclass", func() {
			err := Client.deleteStorageClass(backendScName)
			Expect(err).To(BeNil(), "while deleting storageclass=%s", backendScName)
		})
	})
})
