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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	corev1 "k8s.io/api/core/v1"
)

/*
 * This test will perform following steps:
 * 1. Create two PVC named pvc1 and pvc2 in application namespace
 * 2. Deploy busybox app1, using pvc1, and app2 using pvc2 in application namespace
 * 2. Delete busybox app1 and app2
 * 4. Delete PVC pvc1 and pvc2
 */

var _ = Describe("TEST MULTIPLE NFS SERVER", func() {
	var (
		accessModes          = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity             = "2Gi"
		applicationNamespace = "nfs-tests-ns"

		app1           = "busybox-nfs-1"
		pvcName1       = "pvc-nfs-1"
		label1         = "demo=nfs-deployment-1"
		labelselector1 = map[string]string{
			"demo": "nfs-deployment-1",
		}

		app2           = "busybox-nfs-2"
		label2         = "demo=nfs-deployment-2"
		pvcName2       = "pvc-nfs-2"
		labelselector2 = map[string]string{
			"demo": "nfs-deployment-2",
		}

		OpenEBSNamespace = "openebs"
	)

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create a pvc1", func() {
			var (
				scName = "openebs-rwx"
			)

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName1).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc object %s/%s", applicationNamespace, pvcName1)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName1)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName1)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName1)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName1)
		})

		It("should create a pvc2", func() {
			var (
				scName = "openebs-rwx"
			)

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName2).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc object %s/%s", applicationNamespace, pvcName2)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName2)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName2)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName2)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName2)
		})
	})

	When("nfs-server deployment created", func() {
		It("nfs-server deployment should be created for pvc1", func() {
			By("fetching nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName1)
			Expect(err).ShouldNot(HaveOccurred(), "while fetching pvc %s/%s", applicationNamespace, pvcName1)

			nfsDeployment := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)
			_, err = Client.getDeployment(OpenEBSNamespace, nfsDeployment)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", OpenEBSNamespace, nfsDeployment)
		})

		It("nfs-server deployment should be created for pvc2", func() {
			By("fetching nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName2)
			Expect(err).ShouldNot(HaveOccurred(), "while fetching pvc %s/%s", applicationNamespace, pvcName2)

			nfsDeployment := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)
			_, err = Client.getDeployment(OpenEBSNamespace, nfsDeployment)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", OpenEBSNamespace, nfsDeployment)
		})

	})

	When("deployments with busybox image are created", func() {
		It("should create a app1 deployment and a running pod", func() {
			By("building a deployment")
			deployObj, err := deploy.NewBuilder().
				WithName(app1).
				WithNamespace(applicationNamespace).
				WithLabelsNew(labelselector1).
				WithSelectorMatchLabelsNew(labelselector1).
				WithPodTemplateSpecBuilder(
					pts.NewBuilder().
						WithLabelsNew(labelselector1).
						WithContainerBuildersNew(
							container.NewBuilder().
								WithName("busybox").
								WithImage("busybox").
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
								WithPVCSource(pvcName1),
						),
				).
				Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment object for %s/%s", applicationNamespace, app1)

			By("creating deployment for app1")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment %s/%s", applicationNamespace, app1)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label1, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")
		})

		It("should create a app2 deployment and a running pod", func() {
			By("building a deployment")
			deployObj, err := deploy.NewBuilder().
				WithName(app2).
				WithNamespace(applicationNamespace).
				WithLabelsNew(labelselector2).
				WithSelectorMatchLabelsNew(labelselector2).
				WithPodTemplateSpecBuilder(
					pts.NewBuilder().
						WithLabelsNew(labelselector2).
						WithContainerBuildersNew(
							container.NewBuilder().
								WithName("busybox").
								WithImage("busybox").
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
								WithPVCSource(pvcName2),
						),
				).
				Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment object for %s/%s", applicationNamespace, app2)

			By("creating deployment for app2")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment %s/%s", applicationNamespace, app2)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label2, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("busybox deployments are deleted", func() {
		It("should not have any app1 deployment or running pod", func() {
			By("deleting app1 deployment")
			err := Client.deleteDeployment(applicationNamespace, app1)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", applicationNamespace, app1)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label1, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})

		It("should not have any app2 deployment or running pod", func() {
			By("deleting app2 deployment")
			err := Client.deleteDeployment(applicationNamespace, app2)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", applicationNamespace, app2)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label2, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete pvc1", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName1)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName1)
		})

		It("should delete pvc2", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName2)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName2)
		})
	})
})
