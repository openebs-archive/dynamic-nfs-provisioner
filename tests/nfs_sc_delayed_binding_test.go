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

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
)

/*
 * This test will perform following steps:
 * 1. Create NFS Storageclass with volumeBinding mode WaitForFirstConsumer
 * 2. Create PVC with above storageclass
 * 3. Create busybox deployment with above PVC
 * 4. Delete busybox deployment
 * 5. Delete PVC
 * 6. Delete NFS Storageclass
 */

var _ = Describe("TEST WaitForFirstConsumer binding mode for NFS PV", func() {
	var (
		accessModes          = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity             = "2Gi"
		applicationNamespace = "nfs-tests-ns"

		app           = "busybox-nfs"
		pvcName       = "pvc-nfs"
		label         = "demo=nfs-deployment"
		labelselector = map[string]string{
			"demo": "nfs-deployment",
		}
		appReplica = int32(2)

		NFSScBindingMode = storagev1.VolumeBindingWaitForFirstConsumer
		NFSScName        = "nfs-sc-waitforfirstconsumer"
		backendSCName    = "openebs-hostpath"
		scNfsServerType  = "kernel"
	)

	When("create NFS storageclass", func() {
		It("should create storageclass", func() {
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
			}

			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			err = Client.createStorageClass(&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: NFSScName,
					Annotations: map[string]string{
						string(mayav1alpha1.CASTypeKey):   "nfsrwx",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner:       "openebs.io/nfsrwx",
				VolumeBindingMode: &NFSScBindingMode,
			})
			Expect(err).To(BeNil(), "while creating SC{%s}", NFSScName)
		})
	})

	When(fmt.Sprintf("pvc with storageclass=%s is created", NFSScName), func() {
		It("should create a pvc", func() {
			By("building a pvc")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(NFSScName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc object %s/%s", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When("deployments with busybox image are created", func() {
		It("should create a deployment and a running pod", func() {
			By("building a deployment")
			deployObj, err := deploy.NewBuilder().
				WithName(app).
				WithNamespace(applicationNamespace).
				WithLabelsNew(labelselector).
				WithSelectorMatchLabelsNew(labelselector).
				WithReplicas(&appReplica).
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
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment object for %s/%s", applicationNamespace, app)

			By("creating deployment for app2")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment %s/%s", applicationNamespace, app)

			By(fmt.Sprintf("verifying pod count as %d", appReplica))
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, int(appReplica))
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("busybox deployments are deleted", func() {
		It("should not have any app deployment or running pod", func() {
			By("deleting app deployment")
			err := Client.deleteDeployment(applicationNamespace, app)
			Expect(err).To(BeNil(), "while deleting deployment %s/%s", applicationNamespace, app)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete pvc", func() {
			By("deleting above pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)
		})
	})

	When(fmt.Sprintf("StorageClass %s is deleted", NFSScName), func() {
		It("should delete the SC", func() {
			By("deleting SC")
			err := Client.deleteStorageClass(NFSScName)
			Expect(err).To(BeNil(), "while deleting sc {%s}", NFSScName)
		})
	})

})
