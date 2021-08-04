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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
)

var _ = Describe("TEST VOLUME EVENT MARKING", func() {
	var (
		applicationNamespace = "default"

		// pvc values
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "send-volume-event"

		// nfs provisioner values
		NFSProvisionerName = "openebs-nfs-provisioner"
		openebsNamespace   = "openebs"
		nfsPvName          = ""
		backendPvcName     = ""
		backendPvName      = ""
	)

	When("provisioner deployment updated with VolumeEvent Env", func() {
		It("should update the provisioner deployment", func() {
			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")

			By("updating a deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(
				BeNil(),
				"while fetching deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)

			By("updating the deployment")
			nsEnv := corev1.EnvVar{
				Name:  string(provisioner.NFSVolumeEventsKey),
				Value: "true",
			}

			deploy.Spec.Template.Spec.Containers[0].Env = append(
				deploy.Spec.Template.Spec.Containers[0].Env,
				nsEnv,
			)
			_, err = Client.updateDeployment(deploy)
			Expect(err).To(
				BeNil(),
				"while updating deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)
			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

	When("pvc with storageclass openebs-rwx is created", func() {
		It("should create a pvc ", func() {
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
			Expect(err).To(BeNil(), "while building pvc %s/%s object", applicationNamespace, pvcName)

			By("creating above pvc")
			err = Client.createPVC(pvcObj)
			Expect(err).To(BeNil(), "while creating pvc %s/%s", applicationNamespace, pvcName)

			pvcPhase, err := Client.waitForPVCBound(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while waiting for pvc %s/%s bound phase", applicationNamespace, pvcName)
			Expect(pvcPhase).To(Equal(corev1.ClaimBound), "pvc %s/%s should be in bound phase", applicationNamespace, pvcName)

			nfsPvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			backendPvcName = "nfs-pvc-" + string(nfsPvcObj.UID)
			nfsPvName = nfsPvcObj.Spec.VolumeName

			backendPvcObj, err := Client.getPVC(openebsNamespace, backendPvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", openebsNamespace, pvcName)
			Expect(eventFinalizerExists(&backendPvcObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&backendPvcObj.ObjectMeta)).To(BeFalse(), "volume-event annotation should not be set")

			backendPvName = backendPvcObj.Spec.VolumeName

			backendPvObj, err := Client.getPV(backendPvName)
			Expect(err).To(BeNil(), "while fetching backend PV=%s", backendPvName)
			Expect(eventFinalizerExists(&backendPvObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&backendPvObj.ObjectMeta)).To(BeFalse(), "volume-event annotation should be set")

			nfsPvObj, err := Client.getPV(nfsPvName)
			Expect(err).To(BeNil(), "while fetching NFS PV=%s", nfsPvName)
			Expect(eventFinalizerExists(&nfsPvObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&nfsPvObj.ObjectMeta)).To(BeTrue(), "volume-event annotation should be set")
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should not delete the pvc", func() {
			By("deleting the pvc")
			err := Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc %s/%s", applicationNamespace, pvcName)

			maxRetryCount := 10
			var nfsPvcDeleted bool
			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPVC(applicationNamespace, pvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					nfsPvcDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}
			Expect(nfsPvcDeleted).To(BeTrue(), "NFS pvc should be deleted")

			backendPvcObj, err := Client.getPVC(openebsNamespace, backendPvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", openebsNamespace, backendPvcName)
			Expect(backendPvcObj.DeletionTimestamp).NotTo(BeNil(), "backend PVC should be in Terminating state")
			Expect(eventFinalizerExists(&backendPvcObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&backendPvcObj.ObjectMeta)).To(BeFalse(), "volume-event annotation should not be set")

			backendPvName = backendPvcObj.Spec.VolumeName

			backendPvObj, err := Client.getPV(backendPvName)
			Expect(err).To(BeNil(), "while fetching backend PV=%s", backendPvName)
			Expect(backendPvObj.DeletionTimestamp).To(BeNil(), "backend PV should not be in Terminating state")
			Expect(eventFinalizerExists(&backendPvObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&backendPvObj.ObjectMeta)).To(BeFalse(), "volume-event annotation should not be set")

			nfsPvObj, err := Client.getPV(nfsPvName)
			Expect(err).To(BeNil(), "while fetching NFS PV=%s", nfsPvName)
			Expect(nfsPvObj.DeletionTimestamp).NotTo(BeNil(), "NFS PV should be in Terminating state")
			Expect(eventFinalizerExists(&nfsPvObj.ObjectMeta)).To(BeTrue(), "volume-event finalizer should be set")
			Expect(eventAnnotationExists(&nfsPvObj.ObjectMeta)).To(BeTrue(), "volume-event annotation should be set")
		})
	})

	When("cleaning up backend PV", func() {
		It("should delete backend PV", func() {
			backendPvcObj, err := Client.getPVC(openebsNamespace, backendPvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", openebsNamespace, backendPvcName)
			removeEventFinalizer(&backendPvcObj.ObjectMeta)
			_, err = Client.updatePVC(backendPvcObj)
			Expect(err).To(BeNil(), "while updating pvc %s/%s", openebsNamespace, backendPvcName)

			backendPvObj, err := Client.getPV(backendPvName)
			Expect(err).To(BeNil(), "while fetching backend PV=%s", backendPvName)
			removeEventFinalizer(&backendPvObj.ObjectMeta)
			_, err = Client.updatePV(backendPvObj)
			Expect(err).To(BeNil(), "while updating backend PV=%s", backendPvName)

			nfsPvObj, err := Client.getPV(nfsPvName)
			Expect(err).To(BeNil(), "while fetching NFS PV=%s", nfsPvName)
			removeEventFinalizer(&nfsPvObj.ObjectMeta)
			_, err = Client.updatePV(nfsPvObj)
			Expect(err).To(BeNil(), "while updating NFS PV=%s", nfsPvName)

			maxRetryCount := 10
			var (
				backendPvcDeleted bool
				backendPvDeleted  bool
				nfsPvDeleted      bool
			)

			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPVC(openebsNamespace, backendPvcName)
				if err != nil && k8serrors.IsNotFound(err) {
					backendPvcDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}

			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPV(backendPvName)
				if err != nil && k8serrors.IsNotFound(err) {
					backendPvDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}

			for retries := 0; retries < maxRetryCount; retries++ {
				_, err := Client.getPV(nfsPvName)
				if err != nil && k8serrors.IsNotFound(err) {
					nfsPvDeleted = true
					break
				}
				time.Sleep(time.Second * 5)
			}
			Expect(backendPvcDeleted).To(BeTrue(), "backend pvc should be deleted")
			Expect(backendPvDeleted).To(BeTrue(), "backend PV should be deleted")
			Expect(nfsPvDeleted).To(BeTrue(), "NFS PV should be deleted")
		})
	})

	When("VolumeEvent Env removed from provisioner", func() {
		It("should update the provisioner deployment", func() {

			By("updating a deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(
				BeNil(),
				"while fetching deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)

			By("updating the provisioner deployment")
			idx := 0
			for idx < len(deploy.Spec.Template.Spec.Containers[0].Env) {
				if deploy.Spec.Template.Spec.Containers[0].Env[idx].Name == string(provisioner.NFSVolumeEventsKey) {
					break
				}
				idx++
			}
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env[:idx], deploy.Spec.Template.Spec.Containers[0].Env[idx+1:]...)
			_, err = Client.updateDeployment(deploy)
			Expect(err).To(
				BeNil(),
				"while updating deployment {%s} in namespace {%s}",
				NFSProvisionerName,
				OpenEBSNamespace,
			)

			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

})

func eventFinalizerExists(objMeta *metav1.ObjectMeta) bool {
	eventFinalizer := provisioner.OpenebsEventFinalizerPrefix + provisioner.OpenebsEventFinalizer

	for _, f := range objMeta.Finalizers {
		if f == eventFinalizer {
			return true
		}
	}
	return false
}

func eventAnnotationExists(objMeta *metav1.ObjectMeta) bool {
	for k, v := range objMeta.Annotations {
		if k == provisioner.OpenebsEventAnnotation && v == "true" {
			return true
		}
	}
	return false
}

func removeEventFinalizer(objMeta *metav1.ObjectMeta) {
	eventFinalizer := provisioner.OpenebsEventFinalizerPrefix + provisioner.OpenebsEventFinalizer
	for i, f := range objMeta.Finalizers {
		if f == eventFinalizer {
			objMeta.Finalizers = append(objMeta.Finalizers[:i], objMeta.Finalizers[i+1:]...)
			break
		}
	}
	return
}
