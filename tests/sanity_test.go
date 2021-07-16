/*
Copyright 2019-2020 The OpenEBS Authors

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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("TEST NFS PV", func() {
	var (
		accessModes   = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity      = "2Gi"
		deployName    = "busybox-nfs"
		label         = "demo=nfs-deployment"
		pvcName       = "pvc-nfs"
		labelselector = map[string]string{
			"demo": "nfs-deployment",
		}

		openebsNamespace = "openebs"
		maxRetryCount    = 10
	)

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create a pvc ", func() {
			var (
				scName = "openebs-rwx"
			)

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(
				HaveOccurred(),
				"while building pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

			By("creating above pvc")
			err = Client.createPVC(pvcObj, true)
			Expect(err).To(
				BeNil(),
				"while creating pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)
		})
	})

	When("deployment with busybox image is created", func() {
		It("should create a deployment and a running pod", func() {

			By("building a deployment")
			deployObj, err := deploy.NewBuilder().
				WithName(deployName).
				WithNamespace(applicationNamespace).
				WithLabelsNew(labelselector).
				WithSelectorMatchLabelsNew(labelselector).
				WithPodTemplateSpecBuilder(
					pts.NewBuilder().
						WithLabelsNew(labelselector).
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
								WithPVCSource(pvcName),
						),
				).
				Build()
			Expect(err).ShouldNot(
				HaveOccurred(),
				"while building deployment {%s} in namespace {%s}",
				deployName,
				applicationNamespace,
			)

			By("creating above deployment")
			err = Client.createDeployment(deployObj)
			Expect(err).To(
				BeNil(),
				"while creating deployment {%s} in namespace {%s}",
				deployName,
				applicationNamespace,
			)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

		})
	})

	When("deployment is deleted", func() {
		It("should not have any deployment or running pod", func() {

			By("deleting above deployment")
			err = Client.deleteDeployment(applicationNamespace, deployName)
			Expect(err).To(
				BeNil(),
				"while deleting deployment {%s} in namespace {%s}",
				deployName,
				applicationNamespace,
			)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")

		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete all the NFS services and backend PVC", func() {
			var retries int

			By("fetch PVC information")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc {%s} information in namespace {%s}", pvcName, applicationNamespace)

			By("deleting above pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("verify deletion of NFS-Service service")
			isNFSServiceExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getService(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isNFSServiceExist = false
					break
				}
				Expect(err).To(BeNil(), "while fetching NFS-Server service")
				time.Sleep(time.Second * 5)
			}
			Expect(isNFSServiceExist).To(BeFalse(), "NFS service should not exist after deleting nfs pvc")

			By("verify deletion of NFS-Server instance")
			nfsServerLabels := "openebs.io/nfs-server=nfs-" + pvcObj.Spec.VolumeName
			err = Client.waitForPods(openebsNamespace, nfsServerLabels, corev1.PodRunning, 0)

			isNFSDeploymentExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getDeployment(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isNFSDeploymentExist = false
					break
				}
				Expect(err).To(BeNil(), "while listing deployments of NFS-Server instance")
				time.Sleep(time.Second * 5)
			}
			Expect(isNFSDeploymentExist).To(BeFalse(), "NFS-Server deployment should not exist after deleting nfs pvc")

			By("verify deletion of backend pvc")
			isBackendPVCExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getPVC(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isBackendPVCExist = false
					break
				}
				Expect(err).To(BeNil(), "while fetching backend pvc")
				time.Sleep(time.Second * 5)
			}
			Expect(isBackendPVCExist).To(BeFalse(), "backend pvc should not exist after deleting nfs pvc")
		})
	})

})
