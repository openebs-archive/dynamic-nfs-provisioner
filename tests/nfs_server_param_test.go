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

var _ = Describe("TEST NFS SERVER CONFIGURATION", func() {
	var (
		// application values
		deployName           = "busybox-nfs"
		label                = "demo=nfs-deployment"
		applicationNamespace = "nfs-tests-ns"
		labelselector        = map[string]string{
			"demo": "nfs-deployment",
		}

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "pvc-nfs-param"

		// nfs provisioner values
		openebsNamespace = "openebs"
		nfsServerLabel   = "openebs.io/nfs-server"
		scName           = "nfs-server-config-sc"
		backendSCName    = "openebs-hostpath"
		scNfsServerType  = "kernel"
		scGraceTime      = "30"
		scLeaseTime      = "30"
		scExportConfig   = "/nfsshare *(rw,fsid=0,async,no_auth_nlm)"
		procNfsGraceFile = "/proc/fs/nfsd/nfsv4gracetime"
		procNfsLeaseFile = "/proc/fs/nfsd/nfsv4leasetime"
		exportFile       = "/etc/exports"
	)

	When("create storageclass with nfs configuration", func() {
		It("should create storageclass", func() {
			By("creating storageclass")

			casObj := []mayav1alpha1.Config{
				{
					Name:  provisioner.KeyPVNFSServerType,
					Value: scNfsServerType,
				},
				{
					Name:  provisioner.LeaseTime,
					Value: scLeaseTime,
				},
				{
					Name:  provisioner.GraceTime,
					Value: scGraceTime,
				},
				{
					Name:  provisioner.CustomServerConfig,
					Value: scExportConfig,
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
			Expect(err).To(BeNil(), "while creating SC{%s}", scName)
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

	When("verifying nfs-server configuration", func() {
		It("should have nfs-server configuration set", func() {
			By("fetching nfs-server deployment name")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).ShouldNot(
				HaveOccurred(),
				"while fetching pvc {%s} in namespace {%s}",
				pvcName,
				applicationNamespace,
			)

			nfsDeployment := fmt.Sprintf("nfs-%s", pvcObj.Spec.VolumeName)
			podList, err := Client.listPods(openebsNamespace, fmt.Sprintf("%s=%s", nfsServerLabel, nfsDeployment))
			Expect(err).To(BeNil(), "while fetching nfs-server pod")

			// check if grace period is set or not
			stdOut, stdErr, err := Client.Exec("cat "+procNfsGraceFile,
				podList.Items[0].Name,
				"nfs-server",
				openebsNamespace,
			)
			Expect(err).To(BeNil(), "while reading file=%s err={%s}", procNfsGraceFile, stdErr)
			// remove new line from output
			Expect(stdOut[:len(stdOut)-1]).To(Equal(scGraceTime), "while verifying grace time")

			// check if lease period is set or not
			stdOut, stdErr, err = Client.Exec("cat "+procNfsLeaseFile,
				podList.Items[0].Name,
				"nfs-server",
				openebsNamespace,
			)
			Expect(err).To(BeNil(), "while reading file=%s err={%s}", procNfsLeaseFile, stdErr)
			Expect(stdOut[:len(stdOut)-1]).To(Equal(scLeaseTime), "while verifying lease time")

			// check if export config is set or not
			stdOut, stdErr, err = Client.Exec("cat "+exportFile,
				podList.Items[0].Name,
				"nfs-server",
				openebsNamespace,
			)
			Expect(err).To(BeNil(), "while reading file=%s err={%s}", exportFile, stdErr)
			Expect(stdOut[:len(stdOut)-1]).To(Equal(scExportConfig), "while verifying export config")
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

	When(fmt.Sprintf("StorageClass %s is deleted", scName), func() {
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
