package tests

import (
	"time"

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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
 * Following test will verify:
 * 1. Reclaim Policy behavior of volume
 */

var _ = Describe("TEST NFS VOLUME WITH RECLAIM POLICY", func() {
	var (
		openebsNamespace = "openebs"
		maxRetryCount    = 10

		// SC related options
		scNfsServerType = "kernel"
		backendSCName   = "openebs-hostpath"

		// PVC related options
		accessModes   = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity      = "1Gi"
		scName        = "reclaim-openebs-rwx"
		deployName    = "busybox-reclaim-nfs"
		pvcName       = "reclaim-nfs-pvc"
		claimedPVCObj *corev1.PersistentVolumeClaim

		// Application related options
		label         = "demo=busybox-reclaim-nfs"
		labelselector = map[string]string{
			"demo": "busybox-reclaim-nfs",
		}
		appDeploymentBuilder = deploy.NewBuilder().
					WithName(deployName).
					WithNamespace(applicationNamespace).
					WithLabelsNew(labelselector).
					WithSelectorMatchLabelsNew(labelselector).
					WithStrategyType(appsv1.RecreateDeploymentStrategyType).
					WithPodTemplateSpecBuilder(
				pts.NewBuilder().
					WithLabelsNew(labelselector).
					WithSecurityContext(
						&corev1.PodSecurityContext{
							RunAsUser: func() *int64 {
								var val int64 = 175
								return &val
							}(),
							RunAsGroup: func() *int64 {
								var val int64 = 175
								return &val
							}(),
						},
					).
					WithContainerBuildersNew(
						container.NewBuilder().
							WithName("busybox").
							WithImage("busybox").
							WithCommandNew(
								[]string{
									"/bin/sh",
								},
							).
							WithArgumentsNew(
								[]string{
									"-c",
									"while true ;do sleep 50; done",
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
			)
	)

	When("StorageClass with reclaim policy is created", func() {
		It("should create a StorageClass", func() {
			reclaimPolicy := corev1.PersistentVolumeReclaimRetain
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
						"openebs.io/cas-type":             "nfsrwx",
						string(mayav1alpha1.CASConfigKey): string(casObjStr),
					},
				},
				Provisioner:   "openebs.io/nfsrwx",
				ReclaimPolicy: &reclaimPolicy,
			})
			Expect(err).To(BeNil(), "while creating SC {%s}", scName)
		})
	})

	When("pvc with storageclass "+scName+" is created", func() {
		It("should create a pvc ", func() {

			By("Building PVC")
			pvcObj, err := pvc.NewBuilder().
				WithName(pvcName).
				WithNamespace(applicationNamespace).
				WithStorageClass(scName).
				WithAccessModes(accessModes).
				WithCapacity(capacity).Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("creating PVC")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc {%s} in namespace {%s}", pvcName, applicationNamespace)
		})
	})

	When("busybox deployment is created", func() {
		It("should come into running state", func() {

			By("building a deployment")
			deployObj, err := appDeploymentBuilder.Build()
			Expect(err).ShouldNot(HaveOccurred(), "while building deployment {%s} in namespace {%s}", deployName, applicationNamespace)

			By("creating above deployment")
			err = Client.createDeployment(deployObj)
			Expect(err).To(BeNil(), "while creating deployment {%s} in namespace {%s}", deployName, applicationNamespace)

			By("verifying pod count as 1")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 1)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("deployment is deleted", func() {
		It("should not have any running pod", func() {

			By("deleting above deployment")
			err = Client.deleteDeployment(applicationNamespace, deployName)
			Expect(err).To(BeNil(), "while deleting deployment {%s} in namespace {%s}", deployName, applicationNamespace)

			By("verifying pod count as 0")
			err = Client.waitForPods(applicationNamespace, label, corev1.PodRunning, 0)
			Expect(err).To(BeNil(), "while verifying pod count")
		})
	})

	When("pvc with storageclass reclaim-openebs-rwx is deleted ", func() {
		It("should delete the pvc but not NFS service related artifacts", func() {
			var err error

			// Store bounded PVC object
			claimedPVCObj, err = Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("deleting pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			isPVCDeleted := false
			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPVC(applicationNamespace, pvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					isPVCDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}
			Expect(isPVCDeleted).To(BeTrue(), "pvc should be deleted from cluster")

			pvObj, err := Client.getPV(claimedPVCObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while fetching pv {%s} details", claimedPVCObj.Spec.VolumeName)
			Expect(pvObj.DeletionTimestamp).To(BeNil(), "deletion timestamp shouldn't be set on pv {%s}", claimedPVCObj.Spec.VolumeName)

			backendNFSName := "nfs-" + claimedPVCObj.Spec.VolumeName
			deploymentObj, err := Client.getDeployment(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while fetching deployment {%s} in namespace {%s}", backendNFSName, openebsNamespace)
			Expect(deploymentObj.DeletionTimestamp).To(BeNil(), "deletion timestamp shouldn't be set on deployment {%s}", backendNFSName)

			svcObj, err := Client.getService(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while fetching service {%s} in namespace {%s}", backendNFSName, openebsNamespace)
			Expect(svcObj.DeletionTimestamp).To(BeNil(), "deletion timestamp shouldn't be set on service {%s}", backendNFSName)

			backendPVCObj, err := Client.getPVC(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while fetching backend pvc {%s} in namespace {%s}", backendNFSName, openebsNamespace)
			Expect(backendPVCObj.DeletionTimestamp).To(BeNil(), "deletion timestamp shouldn't be set on backend pvc {%s}", backendNFSName)

		})
	})

	When("deleting NFS service related resources", func() {
		It("should get deleted", func() {
			Expect(claimedPVCObj).NotTo(BeNil(), "claimed pvc shouldn't be nil")
			backendNFSName := "nfs-" + claimedPVCObj.Spec.VolumeName

			err := Client.deleteService(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while deleting service {%s} in namespace {%s}", backendNFSName, openebsNamespace)

			err = Client.deleteDeployment(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while deleting deployment {%s} in namespace {%s}", backendNFSName, openebsNamespace)

			err = Client.deletePVC(openebsNamespace, backendNFSName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", backendNFSName, openebsNamespace)

			err = Client.deletePV(claimedPVCObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while deleting pv {%s}", claimedPVCObj.Spec.VolumeName)
		})
	})

	When("reclaim-openebs-rwx StorageClass is deleted ", func() {
		It("should delete the SC", func() {

			By("deleting SC")
			err = Client.deleteStorageClass(scName)
			Expect(err).To(BeNil(), "while deleting sc {%s}", scName)
		})
	})
})
