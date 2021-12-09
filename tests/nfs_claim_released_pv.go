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

	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
)

var _ = Describe("TEST RECLAIM OF RELEASED NFS PV", func() {
	var (
		// application parameters
		applicationNamespace = "default"
		appName              = "busybox-nfs"
		appLabel             = "demo=busybox-deployment"
		appLabelSelector     = map[string]string{
			"demo": "busybox-deployment",
		}
		// appFileName created in application volume
		appFileName = ""

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "reclaim-released-pv"

		// nfs provisioner values
		openebsNamespace = "openebs"
		scName           = "nfs-pv-sc-retain"
		backendSCName    = "openebs-hostpath"
		scReclaimPolicy  = corev1.PersistentVolumeReclaimRetain
		scNfsServerType  = "kernel"
		// nfsPv stores pv name created by application pvc
		nfsPv = ""
	)

	When("create storageclass with Reclaim policy set to Retain", func() {
		It("should create storageclass", func() {
			By("creating storageclass")
			casObj := []mayav1alpha1.Config{
				{
					Name:  provisioner.KeyPVNFSServerType,
					Value: scNfsServerType,
				},
				{
					Name:  provisioner.KeyPVBackendStorageClass,
					Value: backendSCName,
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
				Provisioner:   "openebs.io/nfsrwx",
				ReclaimPolicy: &scReclaimPolicy,
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

			_, err = Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting %s/%s pvc to bound", applicationNamespace, pvcName)

			By("checking backend PVC bound state")
			pvcObj, err = Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			nfsPv = pvcObj.Spec.VolumeName
			backendPVCName := "nfs-" + nfsPv

			_, err = Client.waitForPVCBound(openebsNamespace, backendPVCName)
			Expect(err).To(BeNil(), "while waiting %s/%s pvc to bound", openebsNamespace, backendPVCName)
		})
	})

	When("creating a application deployment and writing data", func() {
		It("should create a deployment and a running pod", func() {
			By("building a deployment")
			deployObj, err := deploy.NewBuilder().
				WithName(appName).
				WithNamespace(applicationNamespace).
				WithLabelsNew(appLabelSelector).
				WithSelectorMatchLabelsNew(appLabelSelector).
				WithPodTemplateSpecBuilder(
					pts.NewBuilder().
						WithLabelsNew(appLabelSelector).
						WithContainerBuildersNew(
							container.NewBuilder().
								WithName("busybox").
								WithImage("busybox").
								WithImagePullPolicy(corev1.PullIfNotPresent).
								WithCommandNew(
									[]string{
										"sleep",
										"3600",
									},
								).
								WithVolumeMountsNew(
									[]corev1.VolumeMount{
										{
											Name:      "demo-vol1",
											MountPath: "/mnt/store1",
										},
									},
								),
						).
						WithVolumeBuilders(
							volume.NewBuilder().
								WithName("demo-vol1").
								WithPVCSource(pvcName),
						),
				).
				Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment object")

			By("creating above deployment")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment %s/%s", applicationNamespace, appName)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, appLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

			podList, err := Client.listPods(applicationNamespace, appLabel)
			Expect(err).To(BeNil(), "while fetching nfs-server pod")

			appFileName = "/mnt/store1/" + time.Now().Format("20060102150405")
			_, stdErr, err := Client.Exec([]string{"/bin/sh", "-c", "touch " + appFileName}, podList.Items[0].Name, "busybox", applicationNamespace)
			Expect(err).To(BeNil(), "while creating a file=%s err={%s}", appFileName, stdErr)
		})
	})

	When("application deployment is scaled down", func() {
		It("should scaled down the application", func() {
			By("scale down application deployment")
			deployObj, err := Client.getDeployment(applicationNamespace, appName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", applicationNamespace, appName)

			replicaCount := int32(0)
			deployObj.Spec.Replicas = &replicaCount

			_, err = Client.updateDeployment(deployObj)
			Expect(err).To(BeNil(), "while updating the deployment %s/%s with replicaCount=%d", applicationNamespace, appName, replicaCount)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, appLabel, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			Expect(nfsPv).NotTo(BeNil(), "NFS PV name should not be empty")

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

			pvObj, err := Client.getPV(nfsPv)
			Expect(err).To(BeNil(), "while fetching nfs pv")
			Expect(pvObj.Status.Phase).To(Equal(corev1.VolumeReleased), "while verifying NFS PV bound phase")

			By(fmt.Sprintf("Removing claimRef from pv=%s", nfsPv))
			pvObj.Spec.ClaimRef = nil
			_, err = Client.updatePV(pvObj)
			Expect(err).To(BeNil(), "while updating pv")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is created with volume name", scName), func() {
		It("should create the pvc", func() {
			Expect(nfsPv).NotTo(BeNil(), "NFS PV name should not be empty")

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithVolumeName(nfsPv).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).To(BeNil(), "while building pvc %s/%s object", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)

			_, err = Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting %s/%s pvc to bound", applicationNamespace, pvcName)
		})
	})

	When("scaling up application and verifying data", func() {
		It("should scale up the application and verify the data", func() {
			By("scale up application deployment")
			deployObj, err := Client.getDeployment(applicationNamespace, appName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", applicationNamespace, appName)

			replicaCount := int32(1)
			deployObj.Spec.Replicas = &replicaCount

			_, err = Client.updateDeployment(deployObj)
			Expect(err).To(BeNil(), "while updating the deployment %s/%s with replicaCount=%d", applicationNamespace, appName, replicaCount)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, appLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

			podList, err := Client.listPods(applicationNamespace, appLabel)
			Expect(err).To(BeNil(), "while fetching nfs-server pod")

			_, stdErr, err := Client.Exec([]string{"/bin/sh", "-c", "ls " + appFileName}, podList.Items[0].Name, "busybox", applicationNamespace)
			Expect(err).To(BeNil(), "while checking a file=%s err={%s}", appFileName, stdErr)
		})
	})

	When("busybox deployment is deleted", func() {
		It("should not have any busybox deployment or running pod", func() {
			By("deleting busybox deployment")
			err := Client.deleteDeployment(applicationNamespace, appName)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", applicationNamespace, appName)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, appLabel, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			By(fmt.Sprintf("pvc with storageclass %s is deleted", scName))
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("cleaning up nfs volume", func() {
		It("should delete nfs volume and its resources", func() {
			Expect(nfsPv).NotTo(BeNil(), "NFS PV name should not be empty")

			backendPVCName := "nfs-" + nfsPv
			err := Client.deleteService(openebsNamespace, backendPVCName)
			Expect(err).To(BeNil(), "while deleting service %s/%s", openebsNamespace, backendPVCName)

			err = Client.deleteDeployment(openebsNamespace, backendPVCName)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", openebsNamespace, backendPVCName)

			err = Client.deletePVC(openebsNamespace, backendPVCName)
			Expect(err).To(BeNil(), "while deleting backend pvc %s/%s", openebsNamespace, backendPVCName)

			err = Client.deletePV(nfsPv)
			Expect(err).To(BeNil(), "while deleting pv {%s}", nfsPv)
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
