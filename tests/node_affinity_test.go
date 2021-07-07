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

var _ = Describe("TEST NODE AFFINITY FEATURE", func() {
	var (
		openebsNamespace            = "openebs"
		accessModes                 = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		nodeAffinityKeys            = []string{"kubernetes.io/hostname"}
		capacity                    = "2Gi"
		nfsProvisionerLabel         = "openebs.io/component-name=openebs-nfs-provisioner"
		nfsProvisionerContainerName = "openebs-provisioner-nfs"
		pvcName                     = "node-affinity-pvc-nfs"
		nodeAffinityKeyValues       map[string][]string
	)

	When("node affinity environment variable is added", func() {
		It("should be applied", func() {
			nodeList, err := Client.listNodes("")
			Expect(err).To(BeNil(), "failed to list nodes")
			var nodeAffinityAsValue string

			nodeAffinityKeyValues = make(map[string][]string, len(nodeAffinityKeys))
			// Form affinity rules from multiple nodes
			for _, node := range nodeList.Items {
				for _, key := range nodeAffinityKeys {
					if value, isExist := node.Labels[key]; isExist {
						nodeAffinityKeyValues[key] = append(nodeAffinityKeyValues[key], value)
					}
				}
			}

			for key, values := range nodeAffinityKeyValues {
				nodeAffinityAsValue += key + ":["
				for _, value := range values {
					nodeAffinityAsValue += value + ","
				}
				// remove extra comma
				nodeAffinityAsValue = nodeAffinityAsValue[:len(nodeAffinityAsValue)-1]
				nodeAffinityAsValue += "],"
			}
			// remove extra comma and add key as affinity rules
			nodeAffinityAsValue = nodeAffinityAsValue[:len(nodeAffinityAsValue)-1] + ",kubernetes.io/arch"
			nodeAffinityKeyValues["kubernetes.io/arch"] = []string{}

			deploymentList, err := Client.listDeployments(openebsNamespace, nfsProvisionerLabel)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			nfsProvisionerDeployment := deploymentList.Items[0]
			for index, containerDetails := range nfsProvisionerDeployment.Spec.Template.Spec.Containers {
				if containerDetails.Name == nfsProvisionerContainerName {
					nfsProvisionerDeployment.Spec.
						Template.
						Spec.
						Containers[index].Env = append(
						nfsProvisionerDeployment.Spec.
							Template.
							Spec.
							Containers[index].Env,
						corev1.EnvVar{
							Name:  provisioner.NODEAFFINITYKEY,
							Value: nodeAffinityAsValue,
						},
					)
					break
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add NODEAFFINITY env to NFS Provisioner")
		})
	})

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create NFS Server with affinity rules", func() {
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
			Expect(err).To(
				BeNil(),
				"while creating pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

			_, err = Client.waitForPVCBound(pvcObj.Name, pvcObj.Namespace)
			Expect(err).To(BeNil(), "While waiting for PVC to get into bound state")

			boundedPVCObj, err := Client.getPVC(pvcObj.Namespace, pvcObj.Name)
			Expect(err).To(BeNil(), "While fetching bounded PVC")

			nfsServerLabel := "openebs.io/nfs-server=nfs-" + boundedPVCObj.Spec.VolumeName
			err = Client.waitForPods(openebsNamespace, nfsServerLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

			// Get NFS Server deployment
			nfsServerDeployment, err := Client.getDeployment(openebsNamespace, "nfs-"+boundedPVCObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			// Verify propogation of affinity rules
			for _, rules := range nfsServerDeployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				for _, affinityRule := range rules.MatchExpressions {
					values, isExist := nodeAffinityKeyValues[affinityRule.Key]
					Expect(isExist).Should(BeTrue(), "unknown key %s added under node affinity rules", affinityRule.Key)
					if len(values) == 0 {
						Expect(affinityRule.Operator).Should(
							Equal(corev1.NodeSelectorOpExists),
							"operator for key %s should be %s",
							affinityRule.Key,
							corev1.NodeSelectorOpExists,
						)
						Expect(affinityRule.Values).Should(BeNil(), "values should not exist")
					} else {
						Expect(affinityRule.Operator).Should(
							Equal(corev1.NodeSelectorOpIn),
							"operator for key %s should be %s",
							affinityRule.Key,
							corev1.NodeSelectorOpIn,
						)
						Expect(affinityRule.Values).Should(Equal(values), "values should match with affinity rules")
					}
				}
			}
		})
	})

	When("node affinty rules are removed from env", func() {
		It("should remove from the NFS provisioner", func() {
			deploymentList, err := Client.listDeployments(openebsNamespace, nfsProvisionerLabel)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			nfsProvisionerDeployment := deploymentList.Items[0]
			for cIndex, containerDetails := range nfsProvisionerDeployment.Spec.Template.Spec.Containers {
				if containerDetails.Name == nfsProvisionerContainerName {
					envIndex := 0
					for _, envVar := range containerDetails.Env {
						if envVar.Name == provisioner.NODEAFFINITYKEY {
							break
						}
						envIndex++
					}
					nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env = append(nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env[:envIndex], nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env[envIndex+1:]...)
					break
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add NODEAFFINITY env to NFS Provisioner")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {

			By("deleting above pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(
				BeNil(),
				"while deleting pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

		})
	})
})
