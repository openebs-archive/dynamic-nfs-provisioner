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
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TEST NFS SERVER RESOURCE REQUESTS AND LIMITS", func() {
	var (
		// PVC namespace
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "pvc-nfs-resource-requests-limits"

		// SC configuration
		scName                    = "openebs-rwx-resource-requests-limits"
		resourceRequestsAndLimits = &corev1.ResourceRequirements{
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("10Mi"),
				corev1.ResourceCPU:    resource.MustParse("10m"),
			},
		}
		backendSCName   = "openebs-hostpath"
		scNfsServerType = "kernel"

		// application details
		deployName    = "busybox-resource-requests-limits"
		label         = "demo=nfs-deployment"
		labelselector = map[string]string{
			"demo": "nfs-deployment",
		}
	)

	When(fmt.Sprintf("create readwrite many storageclass %s", scName), func() {
		It("should create NFS storageclass", func() {
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
				{
					Name: "NFSServerResourceRequests",
					Value: func() string {
						data, err := yaml.Marshal(resourceRequestsAndLimits.Requests)
						if err != nil {
							panic(fmt.Sprintf("failed to convert to YAML error %v", err))
						}
						return string(data)
					}(),
				},
				{
					Name: "NFSServerResourceLimits",
					Value: func() string {
						data, err := yaml.Marshal(resourceRequestsAndLimits.Limits)
						if err != nil {
							panic(fmt.Sprintf("failed to convert to YAML error %v", err))
						}
						return string(data)
					}(),
				},
			}

			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			err = Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: scName,
					Annotations: map[string]string{
						string(mayav1alpha1.CASTypeKey):   "local",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner: "openebs.io/nfsrwx",
			})
			Expect(err).To(BeNil(), "while creating SC %s", scName)
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
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc %s/%s", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName)
		})
	})

	When("NFS PVC is bounded to PV with resource requests and limits", func() {
		It("nfs-server deployment should be created and should have resource requests and limits", func() {
			By("fetching nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).ShouldNot(HaveOccurred(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			nfsDeploymentName := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)
			nfsDeployment, err := Client.getDeployment(OpenEBSNamespace, nfsDeploymentName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", OpenEBSNamespace, nfsDeploymentName)

			var nfsServerResourceReq *corev1.ResourceRequirements
			for _, container := range nfsDeployment.Spec.Template.Spec.Containers {
				if container.Name == "nfs-server" {
					nfsServerResourceReq = &container.Resources
				}
			}
			Expect(nfsServerResourceReq).NotTo(BeNil(), "resources should exist in NFS server deployment")
			Expect(reflect.DeepEqual(resourceRequestsAndLimits, nfsServerResourceReq)).Should(BeTrue(), "NFS server resource requirements should match %s", cmp.Diff(resourceRequestsAndLimits, nfsServerResourceReq))
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
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment %s/%s", applicationNamespace, deployName)

			By("creating above deployment")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment %s/%s", applicationNamespace, deployName)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")

		})
	})

	When("busybox deployment is deleted", func() {
		It("should not have any busybox deployment or running pod", func() {
			By("deleting busybox deployment")
			err := Client.deleteDeployment(applicationNamespace, deployName)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", applicationNamespace, deployName)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is deleted", scName), func() {
		It("should delete the pvc", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When(fmt.Sprintf("storageclass %s is deleted", scName), func() {
		It("should delete the storageclass", func() {
			err := Client.deleteStorageClass(scName)
			Expect(err).To(BeNil(), "while deleting storageclass: %s", scName)
		})
	})
})
