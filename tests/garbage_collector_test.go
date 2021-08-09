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

var _ = Describe("TEST GARBAGE COLLECTION OF NFS RESOURCES", func() {
	var (
		// application parameters
		applicationNamespace = "default"

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "reclaim-released-pv"
		pvcUID      = ""

		// nfs provisioner values
		nfsProvisionerName  = "openebs-nfs-provisioner"
		nfsProvisionerLabel = "openebs.io/component-name=openebs-nfs-provisioner"
		openebsNamespace    = "openebs"
		scName              = "nfs-sc"
		backendScName       = "nfs-invalid-backend-sc"
		scNfsServerType     = "kernel"
	)

	When("create nfs storageclass with invalid backend storageclass", func() {
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

			pvcObj, err = Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			pvcUID = string(pvcObj.UID)

			By("wait till nfs-server deployment get created")
			var nfsDeploymentCreated bool
			maxRetryCount := 10

			nfsDeploymentName := "nfs-pvc-" + pvcUID
			for maxRetryCount != 0 {
				_, err := Client.getDeployment(openebsNamespace, nfsDeploymentName)
				if err == nil {
					nfsDeploymentCreated = true
					break
				}

				if !k8serrors.IsNotFound(err) {
					fmt.Printf("error fetching nfs-server deployment resource, err=%v\n", err)
				}

				time.Sleep(5 * time.Second)
				maxRetryCount--
			}
			Expect(nfsDeploymentCreated).Should(BeTrue(), "while checking nfs-server deployment creation")
		})
	})

	When("nfs-provisioner is scaled down", func() {
		It("should scaled down the provisioner", func() {
			By("scale down provisioner")
			deployObj, err := Client.getDeployment(openebsNamespace, nfsProvisionerName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", openebsNamespace, nfsProvisionerName)

			replicaCount := int32(0)
			deployObj.Spec.Replicas = &replicaCount

			_, err = Client.updateDeployment(deployObj)
			Expect(err).To(BeNil(), "while updating the deployment %s/%s with replicaCount=%d", openebsNamespace, nfsProvisionerName, replicaCount)

			By("verifying pod count as 0")
			err = Client.waitForPods(openebsNamespace, nfsProvisionerLabel, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			Expect(pvcUID).NotTo(BeEmpty(), "PVC UID should not be empty")

			By(fmt.Sprintf("pvc with storageclass %s is deleted", scName))
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)

			maxRetryCount := 5
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

			By("checking backend PVC")
			backendPvcName := "nfs-pvc-" + pvcUID
			pvcObj, err := Client.getPVC(openebsNamespace, backendPvcName)
			Expect(err).To(BeNil(), "while fetching nfs pv")
			Expect(pvcObj.Status.Phase).To(Equal(corev1.ClaimPending), "while verifying backend PVC claim phase")

			By("checking nfs-server deployment")
			nfsDeploymentName := "nfs-pvc-" + pvcUID
			deployObj, err := Client.getDeployment(openebsNamespace, nfsDeploymentName)
			Expect(err).To(BeNil(), "while fetching nfs-server deployment")
			Expect(deployObj.Status.UnavailableReplicas == 1).To(BeTrue(), "nfs-server pod should not be in ready state")
		})
	})

	When("nfs-provisioner is scaled-up", func() {
		It("should scale-up nfs-provisioner", func() {
			deployObj, err := Client.getDeployment(openebsNamespace, nfsProvisionerName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", openebsNamespace, nfsProvisionerName)

			replicaCount := int32(1)
			deployObj.Spec.Replicas = &replicaCount

			_, err = Client.updateDeployment(deployObj)
			Expect(err).To(BeNil(), "while updating the deployment %s/%s with replicaCount=%d", openebsNamespace, nfsProvisionerName, replicaCount)

			By("verifying pod count as 1")
			err = Client.waitForPods(openebsNamespace, nfsProvisionerLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("nfs-pv stale resources are cleaned-up", func() {
		It("should not find backend PVC and nfs-server deployment", func() {
			Expect(pvcUID).NotTo(BeEmpty(), "PVC UID should not be empty")

			By("checking backend PVC")
			var backendPvcDeleted bool
			backendPvcName := "nfs-pvc-" + pvcUID
			maxRetryCount := 10

			for maxRetryCount != 0 {
				_, err := Client.getPVC(openebsNamespace, backendPvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					backendPvcDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
				maxRetryCount--
			}
			Expect(backendPvcDeleted).To(BeTrue(), "backend pvc should be deleted")

			By("checking nfs-server deployment")
			var nfsDeploymentDeleted bool
			nfsDeploymentName := "nfs-pvc-" + pvcUID
			maxRetryCount = 10

			for maxRetryCount != 0 {
				_, err := Client.getDeployment(openebsNamespace, nfsDeploymentName)
				if err != nil && k8serrors.IsNotFound(err) {
					nfsDeploymentDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
				maxRetryCount--
			}
			Expect(nfsDeploymentDeleted).To(BeTrue(), "nfs-server deployment should be deleted")
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
