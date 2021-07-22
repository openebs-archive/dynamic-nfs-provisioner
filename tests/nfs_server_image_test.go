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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	corev1 "k8s.io/api/core/v1"
)

/*
 * This test will perform following steps:
 * 1. Update OPENEBS_IO_NFS_SERVER_IMG env on NFS Provisioner deployment with earlier version i.e openebs/nfs-server-alpine:0.4.0
 * 2. Create NFS PVC and verify NFS Server image should match to openebs/nfs-server-alpine:0.4.0
 * 3. Delete NFS PVC
 * 4. Rollback changes made to NFS Provisioner deployment
 */

var _ = Describe("TEST NFS SERVER IMAGE CONFIGURATION", func() {
	var (
		openebsNamespace = "openebs"

		// PVC Configuration
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "nfs-server-image-pvc"
		scName      = "openebs-rwx"

		nfsProvisionerLabel         = "openebs.io/component-name=openebs-nfs-provisioner"
		nfsProvisionerContainerName = "openebs-provisioner-nfs"
		nfsServerImage              = "openebs/nfs-server-alpine:ci"
		prevVersionNFSServerImage   = "openebs/nfs-server-alpine:0.4.0"
	)

	When("nfs server image is updated", func() {
		It("should be applied", func() {

			deploymentList, err := Client.listDeployments(openebsNamespace, nfsProvisionerLabel)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			nfsProvisionerDeployment := deploymentList.Items[0]
			isENVImageExist := false

			for index, containerDetails := range nfsProvisionerDeployment.Spec.Template.Spec.Containers {
				if containerDetails.Name == nfsProvisionerContainerName {
					for envIndex, env := range containerDetails.Env {
						if env.Name == string(provisioner.NFSServerImageKey) {
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env[envIndex].Value = prevVersionNFSServerImage
							isENVImageExist = true
							break
						}
					}

					if !isENVImageExist {
						nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env = append(
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env,
							corev1.EnvVar{Name: string(provisioner.NFSServerImageKey), Value: prevVersionNFSServerImage})
						break
					}
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add %s env to NFS Provisioner", provisioner.NFSServerImageKey)
		})
	})

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create NFS Server deployment with configured image name", func() {

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			pvcObj, err = Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			nfsServerLabelSelector := "openebs.io/nfs-server=nfs-" + pvcObj.Spec.VolumeName
			err = Client.waitForPods(openebsNamespace, nfsServerLabelSelector, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying NFS Server running status")

			nfsServerPodList, err := Client.listPods(openebsNamespace, nfsServerLabelSelector)
			Expect(err).To(BeNil(), "while listing NFS Server pods")

			var containerDetails corev1.Container
			for _, pod := range nfsServerPodList.Items {
				for _, container := range pod.Spec.Containers {
					if container.Name == "nfs-server" {
						containerDetails = container
						break
					}
				}
			}
			Expect(containerDetails.Image).Should(Equal(prevVersionNFSServerImage), "image name should match")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {

			By("deleting above pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)
		})
	})

	When("NFS Server image env is updated to ci image", func() {
		It("should be applied", func() {
			deploymentList, err := Client.listDeployments(openebsNamespace, nfsProvisionerLabel)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")
			nfsProvisionerDeployment := deploymentList.Items[0]
			isENVImageExist := false

			for index, containerDetails := range nfsProvisionerDeployment.Spec.Template.Spec.Containers {
				if containerDetails.Name == nfsProvisionerContainerName {
					for envIndex, env := range containerDetails.Env {
						if env.Name == string(provisioner.NFSServerImageKey) {
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env[envIndex].Value = nfsServerImage
							isENVImageExist = true
							break
						}
					}

					if !isENVImageExist {
						nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env = append(
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env,
							corev1.EnvVar{Name: string(provisioner.NFSServerImageKey), Value: nfsServerImage})
						break
					}
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add %s env to NFS Provisioner", provisioner.NFSServerImageKey)
		})
	})
})
