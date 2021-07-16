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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("TEST NODE AFFINITY FEATURE", func() {
	var (
		// PVC related options
		pvcName          = "node-affinity-pvc-nfs"
		accessModes      = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		nodeAffinityKeys = []string{"kubernetes.io/hostname"}
		capacity         = "2Gi"
		scName           = "openebs-rwx"
		maxRetryCount    = 5

		// NFS Provisioner related options
		openebsNamespace            = "openebs"
		nfsProvisionerLabel         = "openebs.io/component-name=openebs-nfs-provisioner"
		nfsProvisionerContainerName = "openebs-provisioner-nfs"
		nodeAffinityKeyValues       map[string][]string
		futureNodeAffinityValue     = "kubernetes.io/nfs-storage-node"
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
							Name:  string(provisioner.NodeAffinityKey),
							Value: nodeAffinityAsValue,
						},
					)
					break
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add %s env to NFS Provisioner", provisioner.NodeAffinityKey)
		})
	})

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create NFS Server with affinity rules", func() {

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
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			boundedPVCObj, err := Client.getPVC(pvcObj.Namespace, pvcObj.Name)
			Expect(err).To(BeNil(), "While fetching bounded PVC")

			nfsServerLabel := "openebs.io/nfs-server=nfs-" + boundedPVCObj.Spec.VolumeName
			err = Client.waitForPods(openebsNamespace, nfsServerLabel, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while waiting for NFS-Server to come into running state")

			// Get NFS Server deployment
			nfsServerDeployment, err := Client.getDeployment(openebsNamespace, "nfs-"+boundedPVCObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			Expect(nfsServerDeployment.Spec.Template.Spec.Affinity).NotTo(
				BeNil(),
				"affinity should exist on NFS-Server deployment {%s} in namespace {%s}",
				nfsServerDeployment.Name,
				nfsServerDeployment.Namespace,
			)
			Expect(nfsServerDeployment.Spec.Template.Spec.Affinity.NodeAffinity).NotTo(
				BeNil(),
				"node affinity should exist on NFS-Server deployment {%s} in namespace {%s}",
				nfsServerDeployment.Name,
				nfsServerDeployment.Namespace,
			)
			Expect(nfsServerDeployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(
				BeNil(),
				"requiredDuringSchedulingIgnoreDuringExecution should exist on NFS-Server deployment {%s} in namespace {%s}",
				nfsServerDeployment.Name,
				nfsServerDeployment.Namespace,
			)

			// Verify propagation of affinity rules
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

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {

			By(fmt.Sprintf("deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace))
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			for {
				_, err = Client.getPVC(applicationNamespace, pvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					break
				}
				fmt.Printf("Waiting for PVC {%s} in namespace {%s} to get delete \n", pvcName, applicationNamespace)
				time.Sleep(time.Second * 2)
			}
		})
	})

	When("Non existing node affinity environment variable is added", func() {
		It("should be applied", func() {

			deploymentList, err := Client.listDeployments(openebsNamespace, nfsProvisionerLabel)
			Expect(err).To(BeNil(), "failed to list NFS Provisioner deployments")

			nfsProvisionerDeployment := deploymentList.Items[0]
			for index, containerDetails := range nfsProvisionerDeployment.Spec.Template.Spec.Containers {
				if containerDetails.Name == nfsProvisionerContainerName {
					// Since node affinity env already exist let's update value
					isEnvFound := false
					for envIndex := range containerDetails.Env {
						if containerDetails.Env[envIndex].Name == string(provisioner.NodeAffinityKey) {
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env[envIndex].Value = futureNodeAffinityValue
							isEnvFound = true
							break
						}
					}

					// In case if env doesn't exist
					if !isEnvFound {
						nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env = append(
							nfsProvisionerDeployment.Spec.Template.Spec.Containers[index].Env,
							corev1.EnvVar{
								Name:  string(provisioner.NodeAffinityKey),
								Value: futureNodeAffinityValue,
							},
						)
					}
					break
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add %s env to NFS Provisioner", provisioner.NodeAffinityKey)
		})
	})

	When("pvc with storageclass openebs-rwx and invalid nodeaffinity is created", func() {
		It("should not provision volume", func() {

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("creating above pvc")
			err = Client.createPVC(pvcObj, false)
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			pvcObj, err = Client.getPVC(pvcObj.Namespace, pvcObj.Name)
			Expect(err).To(BeNil(), "while fetching PVC %s in namespace %s", pvcObj.Name, pvcObj.Namespace)

			var isExpectedEventExist bool
			for retries := 0; retries < maxRetryCount; retries++ {

				// Verify for provision failure events on PVC
				eventList, err := Client.getEvents(pvcObj)
				Expect(err).To(BeNil(), "while getting PVC %s events in namespace %s", pvcObj.Name, pvcObj.Namespace)

				for _, event := range eventList.Items {
					if event.Reason == "ProvisioningFailed" &&
						strings.Contains(event.Message, provisioner.NodeAffinityRulesMismatchEvent) {
						isExpectedEventExist = true
						break
					}
				}
				if isExpectedEventExist {
					break
				}
				// Event is generating after 20 seconds
				time.Sleep(time.Second * 15)
			}
			Expect(isExpectedEventExist).To(BeTrue(), "node affinity rules mismatch event should exist")
		})
	})

	When("node is labeled with affinity rule", func() {
		It("should be applied and volume should get bound", func() {
			nodeList, err := Client.listNodes("")
			Expect(err).To(BeNil(), "failed to list nodes")

			nodeObj := nodeList.Items[0]
			nodeObj.ObjectMeta.Labels[futureNodeAffinityValue] = "true"
			_, err = Client.updateNode(&nodeObj)
			Expect(err).To(BeNil(), "while updating node with %s label", futureNodeAffinityValue)

			_, err = Client.waitForPVCBound(pvcName, applicationNamespace)
			Expect(err).To(BeNil(), "while waiting for PVC to get bound")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {

			By("deleting above pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			for {
				_, err = Client.getPVC(applicationNamespace, pvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					break
				}
				fmt.Printf("Waiting for PVC  %s/%s to get delete\n", applicationNamespace, pvcName)

				time.Sleep(time.Second * 2)
			}

		})
	})

	When("node is unlabeled the affinity rule", func() {
		It("should be removed", func() {
			nodeList, err := Client.listNodes("")
			Expect(err).To(BeNil(), "failed to list nodes")

			for _, nodeObj := range nodeList.Items {
				nodeObj := nodeObj
				_, isLabelExist := nodeObj.GetLabels()[futureNodeAffinityValue]
				if isLabelExist {
					delete(nodeObj.ObjectMeta.Labels, futureNodeAffinityValue)
					_, err = Client.updateNode(&nodeObj)
					Expect(err).To(BeNil(), "while updating node with %s label", futureNodeAffinityValue)
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
						if envVar.Name == string(provisioner.NodeAffinityKey) {
							break
						}
						envIndex++
					}
					nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env = append(nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env[:envIndex], nfsProvisionerDeployment.Spec.Template.Spec.Containers[cIndex].Env[envIndex+1:]...)
					break
				}
			}

			err = Client.applyDeployment(&nfsProvisionerDeployment)
			Expect(err).To(BeNil(), "failed to add %s env to NFS Provisioner", provisioner.NodeAffinityKey)
		})
	})

})
