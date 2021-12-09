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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TEST NON ROOT USER ACCESSING NFS VOLUME", func() {
	var (
		accessModes      = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		openebsNamespace = "openebs"
		capacity         = "1Gi"
		scName           = "non-root-openebs-rwx"
		deployName       = "busybox-non-root-nfs"
		label            = "demo=non-root-nfs-deployment"
		pvcName          = "non-root-nfs-pvc"
		scNFSCASConfig   = `- name: NFSServerType
  value: "kernel"
- name: BackendStorageClass
  value: "openebs-hostpath"
- name: FSGID
  value: "120"`
		labelselector = map[string]string{
			"demo": "non-root-nfs-deployment",
		}
		appDeploymentBuilder = deploy.NewBuilder().
					WithName(deployName).
					WithNamespace(applicationNamespace).
					WithLabelsNew(labelselector).
					WithSelectorMatchLabelsNew(labelselector).
					WithStrategyType(appsv1.RecreateDeploymentStrategyType).
					WithPodTemplateSpecBuilder(
				pts.NewBuilder().
					WithLabelsNew(labelselector).
					WithSecurityContext(
						&corev1.PodSecurityContext{
							RunAsUser: func() *int64 {
								var val int64 = 175
								return &val
							}(),
							RunAsGroup: func() *int64 {
								var val int64 = 175
								return &val
							}(),
						},
					).
					WithContainerBuildersNew(
						container.NewBuilder().
							WithName("busybox").
							WithImage("busybox").
							WithCommandNew(
								[]string{
									"/bin/sh",
								},
							).
							WithArgumentsNew(
								[]string{
									"-c",
									"while true ;do sleep 50; done",
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
			)
	)

	When("StorageClass with FSGID is created", func() {
		It("should create a StorageClass", func() {
			err := Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: scName,
					Annotations: map[string]string{
						"openebs.io/cas-type":   "nfsrwx",
						"cas.openebs.io/config": scNFSCASConfig,
					},
				},
				Provisioner: "openebs.io/nfsrwx",
			})

			Expect(err).To(BeNil(), "while creating SC{%s}", scName)
		})
	})

	When("pvc with storageclass non-root-openebs-rwx is created", func() {
		It("should create a pvc ", func() {

			By("Building PVC")
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

			By("creating PVC")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName)
		})
	})

	// FIXME: This is a workaround to fix localpv permissions issue
	When("NFS volume permissions are updated", func() {
		It("volume should have corresponding permissions", func() {
			By("update NFS volume permissions")

			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(
				BeNil(),
				"while fetching pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

			nfsServerLabel := "openebs.io/nfs-server=nfs-" + pvcObj.Spec.VolumeName
			// Wait for pod to come into running state
			err = Client.waitForPods(openebsNamespace, nfsServerLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while waiting for NFS Server to come into running state")

			listPods, err := Client.listPods(openebsNamespace, nfsServerLabel)
			Expect(err).To(BeNil(), "while fetching NFS Server details")

			_, stdErr, err := Client.Exec([]string{"/bin/sh", "-c", "chmod o=rx /nfsshare"}, listPods.Items[0].Name, "nfs-server", openebsNamespace)
			Expect(err).To(BeNil(), "while updating permissions of NFS volume stderror {%s}", stdErr)
		})
	})

	When("deployment is created without suplemental groups", func() {
		It("shouldn't be able to access NFS volume", func() {

			By("building a deployment")
			deployObj, err := appDeploymentBuilder.Build()
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

			// Exec and perform wrte operation on mount point
			podList, err := Client.listPods(applicationNamespace, label)
			Expect(err).To(BeNil(), "while listing pods")
			By("Accesing volume via exec API")

			stdOut, stdError, err := Client.Exec(
				[]string{"/bin/sh", "-c", "touch /mnt/store1/testvolume"},
				podList.Items[0].Name,
				"busybox",
				applicationNamespace)
			fmt.Printf(
				"When non root application tried to access volume "+
					"without suplemental groups stdout: {%s} stderr: {%s} error: {%s}",
				stdOut,
				stdError,
				err.Error(),
			)
			Expect(stdError).NotTo(
				BeNil(),
				"non root application without suplemental groups shouldn't access the volume",
			)
			Expect(stdError).Should(
				ContainSubstring("Permission denied"),
				"non root application without suplemental groups shouldn't access the volume",
			)
		})
	})

	When("deployment is updated with suplemental groups", func() {
		It("should be able to access NFS volume", func() {

			By("update deployment with suplemental groups")
			deployObj, err := appDeploymentBuilder.Build()
			Expect(err).ShouldNot(
				HaveOccurred(),
				"while building deployment {%s} in namespace {%s}",
				deployName,
				applicationNamespace,
			)
			// Update deployment with suplemental groups
			deployObj.Spec.
				Template.
				Spec.
				SecurityContext.
				SupplementalGroups = append(deployObj.
				Spec.
				Template.
				Spec.
				SecurityContext.
				SupplementalGroups, []int64{120}...)

			By("applying above deployment")
			err = Client.applyDeployment(deployObj)
			Expect(err).To(
				BeNil(),
				"while patching deployment {%s} in namespace {%s}",
				deployName,
				applicationNamespace,
			)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

			// Exec and perform wrte operation on mount point
			podList, err := Client.listPods(applicationNamespace, label)
			Expect(err).To(BeNil(), "while listing pods")
			By("Accesing volume via exec API")
			stdOut, stdError, err := Client.Exec(
				[]string{"/bin/sh", "-c", "touch /mnt/store1/testvolume"},
				podList.Items[0].Name,
				"busybox",
				applicationNamespace)

			fmt.Printf(
				"When non root application tried to access volume "+
					"with suplemental groups stdout: {%s} stderr: {%s} error: {%v}",
				stdOut,
				stdError,
				err,
			)
			Expect(err).To(
				BeNil(),
				"non root application with suplemental group should access the volume",
			)
			Expect(stdError).To(
				BeEmpty(),
				"non root application with suplemental groups should access the volume",
			)
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

	When("pvc with storageclass non-root-openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {

			By("deleting pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(
				BeNil(),
				"while deleting pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)
		})
	})

	When("non-root-openebs-rwx StorageClass is deleted ", func() {
		It("should delete the SC", func() {

			By("deleting SC")
			err = Client.deleteStorageClass(scName)
			Expect(err).To(
				BeNil(),
				"while deleting sc {%s}",
				scName,
			)
		})
	})
})
