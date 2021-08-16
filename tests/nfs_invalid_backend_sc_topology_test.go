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
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TEST PROVISIONING WITH DIFFERENT TOPOLOGY FOR BACKEND SC", func() {
	var (
		// PVC related options
		applicationNamespace = "default"
		pvcName              = "different-node-affinity-pvc"
		accessModes          = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		nodeAffinityKeys     = []string{"kubernetes.io/hostname"}
		capacity             = "2Gi"
		scName               = "nfs-sc-different-node-affinity"

		// NFS Provisioner related options
		openebsNamespace            = "openebs"
		nfsProvisionerLabel         = "openebs.io/component-name=openebs-nfs-provisioner"
		nfsProvisionerContainerName = "openebs-provisioner-nfs"
		nodeAffinityKeyValues       map[string][]string
		backendScName               = "different-node-affinity"
		backendScNodeAffinityLabel  = "invalid.io/invalid-nfs-label"
		backendPVCName              = ""
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

	When(fmt.Sprintf("create backend storageclass %s with nodeAffinityLabel=%s", backendScName, backendScNodeAffinityLabel), func() {
		It("should create backend storageclass", func() {
			By("creating storageclass")
			casObj := []mayav1alpha1.Config{
				{
					Name:  "StorageType",
					Value: "hostpath",
				},
				{
					Name:  "BasePath",
					Value: "/tmp/openebs",
				},
				{
					// Ref: https://github.com/openebs/dynamic-localpv-provisioner/blob/develop/cmd/provisioner-localpv/app/config.go#L103
					Name:  "NodeAffinityLabel",
					Value: backendScNodeAffinityLabel,
				},
			}

			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			err = Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: backendScName,
					Annotations: map[string]string{
						string(mayav1alpha1.CASTypeKey):   "local",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner: "openebs.io/local",
			})
			Expect(err).To(BeNil(), "while creating SC{%s}", scName)
		})
	})

	When(fmt.Sprintf("create storageclass with backendStorageclass=%s", backendScName), func() {
		It("should create storageclass", func() {
			By("creating storageclass")
			casObj := []mayav1alpha1.Config{
				{
					Name:  provisioner.KeyPVNFSServerType,
					Value: "kernel",
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

	When(fmt.Sprintf("pvc with storageclass=%s is created", scName), func() {
		It("should create NFS Server with affinity rules", func() {
			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc object %s/%s", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("verifying events for PVC", func() {
		It("should have event for timeout error", func() {
			By("fetching a pvc")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).ShouldNot(HaveOccurred(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			backendPVCName = "nfs-pvc-" + string(pvcObj.UID)

			maxRetry := 25
			retryPeriod := 5 * time.Second
			var foundProvisioningFailedEvent bool
			for maxRetry != 0 && !foundProvisioningFailedEvent {
				events, err := Client.listEvents(applicationNamespace)
				Expect(err).To(BeNil(), "while fetching events for namespace {%s}", applicationNamespace)

				for _, cn := range events.Items {
					if strings.Contains(cn.Message, fmt.Sprintf("timed out waiting for PVC{openebs/nfs-pvc-%s", pvcObj.UID)) {
						foundProvisioningFailedEvent = true
						break
					}
				}
				time.Sleep(retryPeriod)
				maxRetry--
			}
			Expect(foundProvisioningFailedEvent).Should(BeTrue(), "while checking for ProvisioningFailed event")
		})
		It("should have event for backendPVC", func() {
			maxRetry := 25
			retryPeriod := 5 * time.Second
			var foundProvisioningFailedEvent bool
			for maxRetry != 0 && !foundProvisioningFailedEvent {
				events, err := Client.listEvents(openebsNamespace)
				Expect(err).To(BeNil(), "while fetching events for namespace {%s}", openebsNamespace)

				for _, cn := range events.Items {
					if strings.Contains(cn.Message,
						fmt.Sprintf("failed to provision volume with StorageClass \"%s\": configuration error, no node was specified", backendScName)) {
						if cn.InvolvedObject.Name == backendPVCName {
							foundProvisioningFailedEvent = true
							break
						}
					}
				}
				time.Sleep(retryPeriod)
				maxRetry--
			}
			Expect(foundProvisioningFailedEvent).Should(BeTrue(), "while checking for ProvisioningFailed event")
		})
		It("should have event for nfs-server pod", func() {
			nfsServerLabelSelector := "openebs.io/nfs-server=" + backendPVCName
			nfsServerPodList, err := Client.listPods(openebsNamespace, nfsServerLabelSelector)
			Expect(err).To(BeNil(), "while listing NFS Server pods")

			podName := nfsServerPodList.Items[0].Name

			maxRetry := 25
			retryPeriod := 5 * time.Second
			var foundFailedSchedulingEvent bool
			for maxRetry != 0 && !foundFailedSchedulingEvent {
				events, err := Client.listEvents(openebsNamespace)
				Expect(err).To(BeNil(), "while fetching events for namespace {%s}", openebsNamespace)

				for _, cn := range events.Items {
					if strings.Contains(cn.Message, "0/1 nodes are available: 1 pod has unbound immediate PersistentVolumeClaims.") {
						if cn.InvolvedObject.Name == podName {
							foundFailedSchedulingEvent = true
							break
						}
					}
				}
				time.Sleep(retryPeriod)
				maxRetry--
			}
			Expect(foundFailedSchedulingEvent).Should(BeTrue(), "while checking for FailedScheduling event")
		})
	})

	When(fmt.Sprintf("pvc with storageclass=%s is deleted", scName), func() {
		It("should delete the pvc", func() {
			By(fmt.Sprintf("deleting pvc %s/%s", applicationNamespace, pvcName))
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)

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

	When("node affinity rules are removed from env", func() {
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
	When(fmt.Sprintf("StorageClass %s is deleted", scName), func() {
		It("should delete the storageclass", func() {
			By("deleting storageclass")
			err := Client.deleteStorageClass(scName)
			Expect(err).To(BeNil(), "while deleting sc {%s}", scName)
		})
	})

	When(fmt.Sprintf("backend storageclass %s is deleted", backendScName), func() {
		It("should delete the storageclass", func() {
			err := Client.deleteStorageClass(backendScName)
			Expect(err).To(BeNil(), "while deleting storageclass=%s", backendScName)
		})
	})
})
