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
	"strings"
	"time"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
)

var _ = Describe("TEST NFS PROVISIONER WITH INVALID BACKEND SC", func() {
	var (
		applicationNamespace = "default"

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "pvc-invalid-backend-sc"

		// nfs provisioner values
		scName           = "nfs-server-invalid-sc"
		backendScName    = "nfs-invalid-backend-sc"
		scNfsServerType  = "kernel"
		openebsNamespace = "openebs"
	)

	When("create storageclass with nfs configuration", func() {
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
			Expect(err).To(BeNil(), "while building pvc object for %s/%s", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("verifying NFS PVC state", func() {
		It("should have NFS PVC in pending state", func() {
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)
			Expect(pvcObj.Status.Phase).To(Equal(corev1.ClaimPending), "while verifying NFS PVC claim phase")

			var isExpectedEventExist bool
			maxRetryCount := 15
			backendPvcName := "nfs-pvc-" + pvcObj.UID
			for retries := 0; retries < maxRetryCount; retries++ {
				// Verify for provision failure events on PVC
				eventList, err := Client.getEvents(pvcObj)
				Expect(err).To(BeNil(), "while fetching PVC %s/%s", pvcObj.Namespace, pvcObj.Name)

				for _, event := range eventList.Items {
					if event.Reason == "ProvisioningFailed" &&
						strings.Contains(event.Message,
							fmt.Sprintf("timed out waiting for PVC{%s/%s} to bound", openebsNamespace, backendPvcName)) {
						isExpectedEventExist = true
						break
					}
				}
				if isExpectedEventExist {
					break
				}
				// event will be generated after 60 seconds
				time.Sleep(time.Second * 10)
			}
			Expect(isExpectedEventExist).To(BeTrue(), "ProvisioningFailed event should exist with PVC bound timed out")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When(fmt.Sprintf("StorageClass %s is deleted", scName), func() {
		It("should delete the SC", func() {
			By("deleting SC")
			err := Client.deleteStorageClass(scName)
			Expect(err).To(BeNil(), "while deleting sc {%s}", scName)
		})
	})
})
