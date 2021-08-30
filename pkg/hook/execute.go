/*
Copyright 2021 The OpenEBS Authors.

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

package hook

import (
	"github.com/openebs/dynamic-nfs-provisioner/pkg/helper"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// ExecuteHookOnNFSPV will execute the hook on the given PV and patch it
func (h *Hook) ExecuteHookOnNFSPV(client kubernetes.Interface, pvName string, eventType EventType) error {
	pvObjOrig, err := client.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch PV=%s", pvName)
	}

	pvObj := pvObjOrig.DeepCopy()

	err = h.Action(pvObj, ResourceNFSPV, eventType)
	if err != nil {
		return errors.Wrapf(err, "failed to execute hook")
	}

	data, _, err := helper.GetPatchData(pvObjOrig, pvObj)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().PersistentVolumes().Patch(pvName, types.StrategicMergePatchType, data)
	if err != nil {
		return errors.Wrapf(err, "failed to patch PV=%s", pvObj.Name)
	}

	return nil
}

// ExecuteHookOnBackendPV will execute the hook on the PV for given PVC and patch it
func (h *Hook) ExecuteHookOnBackendPV(client kubernetes.Interface, ns, backendPvcName string, eventType EventType) error {
	pvcObj, err := client.CoreV1().
		PersistentVolumeClaims(ns).
		Get(backendPvcName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch PVC=%s/%s", ns, backendPvcName)
	}

	pvObjOrig, err := client.CoreV1().PersistentVolumes().Get(pvcObj.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch PV=%s", pvcObj.Spec.VolumeName)
	}

	pvObj := pvObjOrig.DeepCopy()
	err = h.Action(pvObj, ResourceBackendPV, eventType)
	if err != nil {
		return errors.Wrapf(err, "failed to execute hook")
	}

	data, _, err := helper.GetPatchData(pvObjOrig, pvObj)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().PersistentVolumes().Patch(pvObj.Name, types.StrategicMergePatchType, data)
	if err != nil {
		return errors.Wrapf(err, "failed to patch PV=%s", pvObj.Name)
	}

	return nil
}

// ExecuteHookOnBackendPV will execute the hook on the PV for given PVC and patch it
func (h *Hook) ExecuteHookOnBackendPVC(client kubernetes.Interface, ns, backendPvcName string, eventType EventType) error {
	pvcObjOrig, err := client.CoreV1().
		PersistentVolumeClaims(ns).
		Get(backendPvcName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch PVC=%s/%s", ns, backendPvcName)
	}

	pvcObj := pvcObjOrig.DeepCopy()

	err = h.Action(pvcObj, ResourceBackendPVC, eventType)
	if err != nil {
		return errors.Wrapf(err, "failed to execute hook")
	}

	data, _, err := helper.GetPatchData(pvcObjOrig, pvcObj)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().PersistentVolumeClaims(ns).Patch(pvcObj.Name, types.StrategicMergePatchType, data)
	if err != nil {
		return errors.Wrapf(err, "failed to patch PVC=%s/%s", ns, backendPvcName)
	}

	return nil
}

// ExecuteHookOnNFSService will execute the hook on the given service and patch it
func (h *Hook) ExecuteHookOnNFSService(client kubernetes.Interface, ns, serviceName string, eventType EventType) error {
	svcObjOrig, err := client.CoreV1().
		Services(ns).
		Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch service=%s/%s", ns, serviceName)
	}

	svcObj := svcObjOrig.DeepCopy()

	err = h.Action(svcObj, ResourceNFSService, eventType)
	if err != nil {
		return errors.Wrapf(err, "failed to execute hook")
	}

	data, _, err := helper.GetPatchData(svcObjOrig, svcObj)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Services(ns).Patch(svcObj.Name, types.StrategicMergePatchType, data)
	if err != nil {
		return errors.Wrapf(err, "failed to patch service=%s/%s", ns, serviceName)
	}

	return nil
}

// ExecuteHookOnNFSDeployment will execute the hook on the given deployment and patch it
func (h *Hook) ExecuteHookOnNFSDeployment(client kubernetes.Interface, ns, deployName string, eventType EventType) error {
	deployObjOrig, err := client.AppsV1().
		Deployments(ns).
		Get(deployName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch deployment=%s/%s", ns, deployName)
	}

	deployObj := deployObjOrig.DeepCopy()

	err = h.Action(deployObj, ResourceNFSServerDeployment, eventType)
	if err != nil {
		return errors.Wrapf(err, "failed to execute hook")
	}

	data, _, err := helper.GetPatchData(deployObjOrig, deployObj)
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Patch(deployObj.Name, types.StrategicMergePatchType, data)
	if err != nil {
		return errors.Wrapf(err, "failed to patch deployment=%s/%s", ns, deployName)
	}

	return nil
}
