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
	deploy "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TEST NFS SERVER FILE PERMISSIONS", func() {
	var (
		// PVC namespace
		accessModes     = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity        = "2Gi"
		pvcName         = "pvc-nfs-resource-requests-limits"
		uid             = "1000"
		gid             = "2000"
		mode            = "0744"
		filePermissions = map[string]string{
			"UID":  uid,
			"GID":  gid,
			"mode": mode,
		}

		// NFS server configuration
		filePermissionsEnvs = map[string]string{
			"FILEPERMISSIONS_UID":  uid,
			"FILEPERMISSIONS_GID":  gid,
			"FILEPERMISSIONS_MODE": mode,
		}
		nfsDeployment *appsv1.Deployment

		// SC configuration
		scName          = "openebs-rwx-file-permissions"
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
			Expect(err).To(BeNil(), "while creating SC %s", scName)
		})
	})

	When(fmt.Sprintf("pvc with storageclass %s is created", scName), func() {
		It("should create a pvc ", func() {
			By("building a pvc")
			casObj := []mayav1alpha1.Config{
				{
					Name: provisioner.FilePermissions,
					Data: filePermissions,
				},
			}
			casObjStr, err := yaml.Marshal(casObj)
			Expect(err).To(BeNil(), "while marshaling cas object")

			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithAnnotations(map[string]string{
					string(mayav1alpha1.CASConfigKey): string(casObjStr),
				}).
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

	When("NFS server is created with backend PV mounted", func() {
		It("should have file permissions ENVs", func() {
			By("generating nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).ShouldNot(HaveOccurred(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			nfsDeploymentName := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)

			By("GETing nfs-server deployment API object")
			nfsDeployment, err = Client.getDeployment(OpenEBSNamespace, nfsDeploymentName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", OpenEBSNamespace, nfsDeploymentName)

			By("checking if FilePermissions ENVs are set correctly in the nfs-server container")
			ok, err := isEnvValueCorrect(&nfsDeployment.Spec.Template.Spec.Containers[0], filePermissionsEnvs)
			Expect(err).To(BeNil(), "while checking if ENV values are set correctly in NFS server container")
			Expect(ok).To(BeTrue(),
				"while checking if the ENV-value set {%v} is"+
					" present in the NFS server container",
				filePermissionsEnvs,
			)
		})

		It("should have correct file permissions on backend vol mountpath", func() {
			By("GETing NFS server Pod from the Deployment")
			podList, err := Client.listDeploymentPods(nfsDeployment)
			Expect(err).To(BeNil(), "when listing the pods of "+
				"NFS server deployment {%s} in namespace {%s}",
				nfsDeployment.Name, nfsDeployment.Namespace,
			)
			Expect(podList.Items).To(Not(BeEmpty()), "when "+
				"listing the pods of NFS server deployment "+
				"{%s} in namespace {%s}",
				nfsDeployment.Name, nfsDeployment.Namespace,
			)

			pod := &podList.Items[0]

			By("exec-ing in the NFS server Pod and checking file permissions")
			stdout, stderr, err := Client.Exec("/bin/bash -c 'stat --printf=%u ${SHARED_DIRECTORY}'", pod.Name, pod.Spec.Containers[0].Name, pod.Namespace)
			Expect(err).To(BeNil(), "when exec-ing into the NFS "+
				"server container in pod {%s} in namespace "+
				"{%s}", pod.Name, pod.Namespace,
			)
			Expect(stderr).To(BeEmpty(), "when checking the "+
				"stderr output of the `stat` command to "+
				"check owner's UID",
			)
			Expect(stdout).To(Equal(uid), "when checking the "+
				"stdout output of the `stat` command to "+
				"check owner's UID",
			)

			stdout, stderr, err = Client.Exec("/bin/bash -c 'stat --printf=%g ${SHARED_DIRECTORY}'", pod.Name, pod.Spec.Containers[0].Name, pod.Namespace)
			Expect(err).To(BeNil(), "when exec-ing into the NFS "+
				"server container in pod {%s} in namespace "+
				"{%s}", pod.Name, pod.Namespace,
			)
			Expect(stderr).To(BeEmpty(), "when checking the "+
				"stderr output of the `stat` command to "+
				"check owner's GID",
			)
			Expect(stdout).To(Equal(gid), "when checking the "+
				"stdout output of the `stat` command to "+
				"check owner's GID",
			)

			stdout, stderr, err = Client.Exec("/bin/bash -c 'stat --printf=%04a ${SHARED_DIRECTORY}'", pod.Name, pod.Spec.Containers[0].Name, pod.Namespace)
			Expect(err).To(BeNil(), "when exec-ing into the NFS "+
				"server container in pod {%s} in namespace "+
				"{%s}", pod.Name, pod.Namespace,
			)
			Expect(stderr).To(BeEmpty(), "when checking the "+
				"stderr output of the `stat` command to "+
				"check the shared directory's file mode",
			)
			Expect(stdout).To(Equal(mode), "when checking the "+
				"stdout output of the `stat` command to "+
				"check the shared directory's file mode",
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
