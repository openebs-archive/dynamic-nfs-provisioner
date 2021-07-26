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

var _ = Describe("TEST CUSTOM NAMESPACE FOR NFS SERVER", func() {
	var (
		accessModes          = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity             = "2Gi"
		deployName           = "busybox-nfs"
		label                = "demo=nfs-deployment"
		pvcName              = "pvc-nfs"
		nfsServerNsEnv       = "OPENEBS_IO_NFS_SERVER_NS"
		OpenEBSNamespace     = "openebs"
		NFSProvisionerName   = "openebs-nfs-provisioner"
		applicationNamespace = "nfs-tests-ns"
		labelselector        = map[string]string{
			"demo": "nfs-deployment",
		}
	)

	When("provisioner deployment updated with NFSServerNamespace", func() {
		It("should update the provisioner deployment", func() {
			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")

			By("updating a deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(
				BeNil(),
				"while fetching deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)

			By("updating the deployment")
			nsEnv := corev1.EnvVar{
				Name:  nfsServerNsEnv,
				Value: applicationNamespace,
			}

			deploy.Spec.Template.Spec.Containers[0].Env = append(
				deploy.Spec.Template.Spec.Containers[0].Env,
				nsEnv,
			)
			_, err = Client.updateDeployment(deploy)
			Expect(err).To(
				BeNil(),
				"while updating deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)
			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

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
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName)
		})
	})

	When("nfs-server deployment created", func() {
		It("nfs-server deployment should be created ", func() {
			By("fetching nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).ShouldNot(
				HaveOccurred(),
				"while fetching pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

			nfsDeployment := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)
			_, err = Client.getDeployment(applicationNamespace, nfsDeployment)
			Expect(err).To(
				BeNil(),
				"while fetching deployment {%s} in namespace {%s}",
				nfsDeployment,
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

	When("busybox deployment is deleted", func() {
		It("should not have any busybox deployment or running pod", func() {
			By("deleting busybox deployment")
			err := Client.deleteDeployment(applicationNamespace, deployName)
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
		It("should delete the pvc", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(
				BeNil(),
				"while deleting pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

		})
	})

	When("NFSServerNamespace removed from provisioner deployment", func() {
		It("should update the provisioner deployment", func() {
			By("fetching provisioner deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(
				BeNil(),
				"while fetching deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)

			By("updating the provisioner deployment")
			idx := 0
			for idx < len(deploy.Spec.Template.Spec.Containers[0].Env) {
				if deploy.Spec.Template.Spec.Containers[0].Env[idx].Name == nfsServerNsEnv {
					break
				}
				idx++
			}
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env[:idx], deploy.Spec.Template.Spec.Containers[0].Env[idx+1:]...)
			_, err = Client.updateDeployment(deploy)
			Expect(err).To(
				BeNil(),
				"while updateingupdating deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)
			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

})
