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
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("TEST INVALID NAMESPACE FOR NFS SERVER", func() {
	var (
		accessModes          = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity             = "2Gi"
		pvcName              = "pvc-nfs"
		nfsServerNsEnv       = "OPENEBS_IO_NFS_SERVER_NS"
		OpenEBSNamespace     = "openebs"
		NFSProvisionerName   = "openebs-nfs-provisioner"
		nfsServerNs          = "nfs-server-invalid-ns"
		applicationNamespace = "nfs-tests-ns"
	)

	When("provisioner deployment updated with NFSServerNamespace", func() {
		It("should update the provisioner deployment", func() {
			By("updating a deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while fetching deployment {%s} in namespace {%s}", NFSProvisionerName, OpenEBSNamespace)

			By("updating the deployment")
			nsEnv := corev1.EnvVar{
				Name:  nfsServerNsEnv,
				Value: nfsServerNs,
			}

			deploy.Spec.Template.Spec.Containers[0].Env = append(
				deploy.Spec.Template.Spec.Containers[0].Env,
				nsEnv,
			)
			_, err = Client.updateDeployment(deploy)
			Expect(err).To(BeNil(), "while updating deployment %s/%s", OpenEBSNamespace, NFSProvisionerName)

			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create a pvc ", func() {
			scName := "openebs-rwx"

			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc %s/%s", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("verifying application PVC state", func() {
		It("should have PVC in pending state", func() {
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)
			Expect(pvcObj.Status.Phase).To(Equal(corev1.ClaimPending), "while verifying PVC claim phase")
		})

		It("should have an event with reason ProvisioningFailed", func() {
			maxRetry := 10
			retryPeriod := 5 * time.Second
			var foundProvisioningFailedEvent bool
			for maxRetry != 0 && !foundProvisioningFailedEvent {
				events, err := Client.listEvents(applicationNamespace)
				Expect(err).To(BeNil(), "while fetching events for namespace {%s}", applicationNamespace)

				for _, cn := range events.Items {
					if strings.Contains(cn.Message, fmt.Sprintf("namespaces \"%s\" not found", nfsServerNs)) {
						foundProvisioningFailedEvent = true
						break
					}
				}
				time.Sleep(retryPeriod)
				maxRetry--
			}
			Expect(foundProvisioningFailedEvent).Should(BeTrue(), "while checking for event ProvisioningFailedEvent")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete the pvc", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("NFSServerNamespace removed from provisioner deployment", func() {
		It("should update the provisioner deployment", func() {
			By("fetching provisioner deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", OpenEBSNamespace, NFSProvisionerName)

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
			Expect(err).To(BeNil(), "while updateingupdating deployment %s/%s", OpenEBSNamespace, NFSProvisionerName)

			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})
})
