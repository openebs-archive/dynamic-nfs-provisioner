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

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	nfshook "github.com/openebs/dynamic-nfs-provisioner/pkg/hook"
	pvc "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	provisioner "github.com/openebs/dynamic-nfs-provisioner/provisioner"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildTestHook() *nfshook.Hook {
	var hook nfshook.Hook
	hook.Config = make(map[nfshook.ActionType]nfshook.HookConfig)
	hook.Config[nfshook.ActionAddOnCreateVolumeEvent] = nfshook.HookConfig{
		Name: "createHook",
		BackendPVConfig: &nfshook.PVHook{
			Annotations: map[string]string{
				"example.io/track": "true",
				"example.io/res":   "backend-pvc",
				"test.io/owner":    "teamA",
			},
			Finalizers: []string{"test.io/tracking-protection"},
		},
		NFSPVConfig: &nfshook.PVHook{
			Annotations: map[string]string{
				"example.io/track": "true",
				"example.io/res":   "nfs-pv",
				"test.io/owner":    "teamA",
			},
			Finalizers: []string{"test.io/tracking-protection"},
		},

		BackendPVCConfig: &nfshook.PVCHook{
			Annotations: map[string]string{
				"example.io/track": "true",
				"example.io/res":   "backend-pvc",
				"test.io/owner":    "teamA",
			},
			Finalizers: []string{"test.io/tracking-protection"},
		},

		NFSServiceConfig: &nfshook.ServiceHook{
			Annotations: map[string]string{
				"example.io/track": "true",
				"example.io/res":   "nfs-svc",
				"test.io/owner":    "teamA",
			},
			Finalizers: []string{"test.io/tracking-protection"},
		},
		NFSDeploymentConfig: &nfshook.DeploymentHook{
			Annotations: map[string]string{
				"example.io/track": "true",
				"example.io/res":   "nfs-deployment",
				"test.io/owner":    "teamA",
			},
			Finalizers: []string{"test.io/tracking-protection"},
		},
	}

	hook.Config[nfshook.ActionRemoveOnDeleteVolumeEvent] = nfshook.HookConfig{
		Name: "deleteHook",
		BackendPVConfig: &nfshook.PVHook{
			Finalizers: []string{"test.io/tracking-protection"},
		},
		NFSPVConfig: &nfshook.PVHook{
			Finalizers: []string{"test.io/tracking-protection"},
		},

		BackendPVCConfig: &nfshook.PVCHook{
			Finalizers: []string{"test.io/tracking-protection"},
		},

		NFSServiceConfig: &nfshook.ServiceHook{
			Finalizers: []string{"test.io/tracking-protection"},
		},
		NFSDeploymentConfig: &nfshook.DeploymentHook{
			Finalizers: []string{"test.io/tracking-protection"},
		},
	}

	hook.Version = nfshook.HookVersion
	return &hook
}

var _ = Describe("TEST NFS HOOK", func() {
	var (
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		capacity    = "2Gi"
		pvcName     = "pvc-nfs"

		backendPVName      = ""
		hookConfigMapName  = "nfs-hook"
		hookVolumeName     = "nfs-hook-vol"
		hook               *nfshook.Hook
		NFSProvisionerName = "openebs-nfs-provisioner"
		openebsNamespace   = "openebs"
		maxRetryCount      = 25
	)

	When("nfs hook configmap is created", func() {
		It("should create a configmap ", func() {
			h := buildTestHook()
			data, err := yaml.Marshal(h)
			Expect(err).To(BeNil(), "while marshalling hook")

			var cmap corev1.ConfigMap
			cmap.Name = hookConfigMapName
			cmap.Namespace = openebsNamespace
			cmap.Data = map[string]string{
				"config": string(data),
			}

			err = Client.createConfigMap(&cmap)
			Expect(err).To(BeNil(), "while creating hook configmap")

			hook = h
		})
	})

	When("provisioner deployment updated with Hook Configmap", func() {
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
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes,
				corev1.Volume{
					Name: hookVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: hookConfigMapName,
							},
						},
					},
				},
			)

			deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts,
				corev1.VolumeMount{
					Name:      hookVolumeName,
					MountPath: provisioner.HookConfigDirectory,
				},
			)

			_, err = Client.updateDeployment(deploy)
			Expect(err).To(BeNil(), "while updating deployment %s/%s", openebsNamespace, NFSProvisionerName)

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

	When("verifying nfs resources", func() {
		It("should have been modified as per hook", func() {
			Expect(hook).NotTo(BeNil(), "hook object should not be nil")

			By("fetch PVC information")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc %s/%s", applicationNamespace, pvcName)

			By("verifying backend PVC")
			backendPVCObj, err := Client.getPVC(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while fetching backend pvc")
			Expect(annotationExist(&backendPVCObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].BackendPVCConfig.Annotations)).
				To(BeTrue(), "Backend PVC=%s/%s should be annotated", openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
			Expect(finalizerExist(&backendPVCObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].BackendPVCConfig.Finalizers)).
				To(BeTrue(), "Backend PVC=%s/%s should be updated with finalizers", openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)

			By("verifying backend PV")
			backendPVName = backendPVCObj.Spec.VolumeName
			backendPV, err := Client.getPV(backendPVName)
			Expect(err).To(BeNil(), "while fetching backend PV")
			Expect(annotationExist(&backendPV.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].BackendPVConfig.Annotations)).
				To(BeTrue(), "Backend PV=%s should be annotated", backendPVName)
			Expect(finalizerExist(&backendPV.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].BackendPVConfig.Finalizers)).
				To(BeTrue(), "Backend PV=%s should be updated with finalizers", backendPVName)

			By("verifying NFS Service")
			svcObj, err := Client.getService(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while fetching NFS Service")
			Expect(annotationExist(&svcObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSServiceConfig.Annotations)).
				To(BeTrue(), "NFS Service=%s/%s should be annotated", openebsNamespace, svcObj.Name)
			Expect(finalizerExist(&svcObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSServiceConfig.Finalizers)).
				To(BeTrue(), "NFS Service=%s/%s should be updated with finalizers", openebsNamespace, svcObj.Name)

			By("verifying NFS Server Deployment")
			deployObj, err := Client.getDeployment(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while fetching NFS Deployment")
			Expect(annotationExist(&deployObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSDeploymentConfig.Annotations)).To(
				BeTrue(),
				"NFS Deployment=%s/%s should be annotated", openebsNamespace, deployObj.Name,
			)
			Expect(finalizerExist(&deployObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSDeploymentConfig.Finalizers)).To(
				BeTrue(),
				"NFS Deployment=%s/%s should be updated with finalizers", openebsNamespace, deployObj.Name,
			)

			By("verifying NFSPV")
			nfsPVObj, err := Client.getPV(pvcObj.Spec.VolumeName)
			Expect(err).To(BeNil(), "while fetching backend PV")
			Expect(annotationExist(&nfsPVObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSPVConfig.Annotations)).
				To(BeTrue(), "NFS PV=%s should be annotated", pvcObj.Spec.VolumeName)
			Expect(finalizerExist(&nfsPVObj.ObjectMeta, hook.Config[nfshook.ActionAddOnCreateVolumeEvent].NFSPVConfig.Finalizers)).
				To(BeTrue(), "NFS PV=%s should be updated with finalizers", pvcObj.Spec.VolumeName)
		})
	})

	When("pvc with storageclass openebs-rwx is deleted ", func() {
		It("should delete all the NFS services and backend PVC", func() {
			Expect(backendPVName).NotTo(BeEmpty(), "backend PV name should not be empty")

			var retries int

			By("fetch PVC information")
			pvcObj, err := Client.getPVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while fetching pvc {%s} information in namespace {%s}", pvcName, applicationNamespace)

			By("deleting above pvc")
			err = Client.deletePVC(applicationNamespace, pvcName)
			Expect(err).To(BeNil(), "while deleting pvc {%s} in namespace {%s}", pvcName, applicationNamespace)

			By("verify deletion of NFS-Service service")
			isNFSServiceExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getService(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isNFSServiceExist = false
					break
				}
				Expect(err).To(BeNil(), "while fetching NFS-Server service")
				time.Sleep(time.Second * 5)
			}
			Expect(isNFSServiceExist).To(BeFalse(), "NFS service should not exist after deleting nfs pvc")

			By("verify deletion of NFS-Server instance")
			nfsServerLabels := "openebs.io/nfs-server=nfs-" + pvcObj.Spec.VolumeName
			err = Client.waitForPods(openebsNamespace, nfsServerLabels, corev1.PodRunning, 0)

			isNFSDeploymentExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getDeployment(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isNFSDeploymentExist = false
					break
				}
				Expect(err).To(BeNil(), "while listing deployments of NFS-Server instance")
				time.Sleep(time.Second * 5)
			}
			Expect(isNFSDeploymentExist).To(BeFalse(), "NFS-Server deployment should not exist after deleting nfs pvc")

			By("verify deletion of backend pvc")
			isBackendPVCExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getPVC(openebsNamespace, "nfs-"+pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isBackendPVCExist = false
					break
				}
				Expect(err).To(BeNil(), "while fetching backend pvc")
				time.Sleep(time.Second * 5)
			}
			Expect(isBackendPVCExist).To(BeFalse(), "backend pvc should not exist after deleting nfs pvc")

			By("verify deletion of NFS PV")
			isNFSPVExist := true
			for retries = 0; retries < maxRetryCount; retries++ {
				_, err = Client.getPV(pvcObj.Spec.VolumeName)
				if err != nil && k8serrors.IsNotFound(err) {
					isNFSPVExist = false
					break
				}
				Expect(err).To(BeNil(), "while fetching NFS PV")
				time.Sleep(time.Second * 5)
			}
			Expect(isNFSPVExist).To(BeFalse(), "NFS PV should not exist after deleting nfs pvc")
		})
	})

	When("Hook configMap removed from provisioner deployment", func() {
		It("should update the provisioner deployment", func() {
			By("fetching provisioner deployment")
			deploy, err := Client.getDeployment(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while fetching deployment %s/%s", openebsNamespace, NFSProvisionerName)

			By("updating the provisioner deployment")
			// Removing volumeMount
			idx := 0
			for idx < len(deploy.Spec.Template.Spec.Containers[0].VolumeMounts) {
				if deploy.Spec.Template.Spec.Containers[0].VolumeMounts[idx].Name == hookVolumeName {
					break
				}
				idx++
			}
			deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts[:idx],
				deploy.Spec.Template.Spec.Containers[0].VolumeMounts[idx+1:]...)

			// Removing volume
			idx = 0
			for idx < len(deploy.Spec.Template.Spec.Volumes) {
				if deploy.Spec.Template.Spec.Volumes[idx].Name == hookVolumeName {
					break
				}
				idx++
			}
			deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes[:idx],
				deploy.Spec.Template.Spec.Volumes[idx+1:]...)

			_, err = Client.updateDeployment(deploy)
			Expect(err).To(BeNil(), "while updating deployment %s/%s", openebsNamespace, NFSProvisionerName)

			By("waiting for deployment rollout")
			err = Client.waitForDeploymentRollout(OpenEBSNamespace, NFSProvisionerName)
			Expect(err).To(BeNil(), "while verifying deployment rollout")
		})
	})

	When("nfs hook configmap is deleted", func() {
		It("should delete a configmap ", func() {
			err := Client.deleteConfigMap(openebsNamespace, hookConfigMapName)
			Expect(err).To(BeNil(), "while deleting hook configmap")
		})
	})
})

func annotationExist(objMeta *metav1.ObjectMeta, annotations map[string]string) bool {
	annExist := true

	for k, v := range annotations {
		if objMeta.Annotations == nil {
			annExist = false
			break
		}

		ev, ok := objMeta.Annotations[k]
		if !ok {
			annExist = false
			break
		}

		if ev != v {
			annExist = false
			break
		}
	}

	return annExist
}

func finalizerExist(objMeta *metav1.ObjectMeta, finalizers []string) bool {
	if len(finalizers) > len(objMeta.Finalizers) {
		return false
	}

	desiredFinalizersMap := make(map[string]struct{})
	for _, f := range finalizers {
		desiredFinalizersMap[f] = struct{}{}
	}

	for _, f := range objMeta.Finalizers {
		delete(desiredFinalizersMap, f)
	}

	if len(desiredFinalizersMap) != 0 {
		return false
	}
	return true
}
