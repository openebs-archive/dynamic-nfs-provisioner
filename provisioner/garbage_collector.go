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

package provisioner

import (
	"context"
	"fmt"
	"time"

	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	errors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	// GarbageCollectorInterval defines periodic interval to run garbage collector
	GarbageCollectorInterval = 5 * time.Minute
)

func RunGarbageCollector(client kubernetes.Interface, pvTracker ProvisioningTracker, ns string, stopCh <-chan struct{}) {
	// NewTicker sends tick only after mentioned interval.
	// So to ensure that the garbage collector gets executed at the beginning,
	// we are running it here.
	klog.V(4).Infof("Running garbage collector for stale NFS resources")
	err := cleanUpStalePvc(client, pvTracker, ns)
	klog.V(4).Infof("Garbage collection completed for stale NFS resources with error=%v", err)

	ticker := time.NewTicker(GarbageCollectorInterval)

	for {
		select {
		case <-stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			klog.V(4).Infof("Running garbage collector for stale NFS resources")
			err = cleanUpStalePvc(client, pvTracker, ns)
			klog.V(4).Infof("Garbage collection completed for stale NFS resources with error=%v", err)
		}
	}
}

func cleanUpStalePvc(client kubernetes.Interface, pvTracker ProvisioningTracker, ns string) error {
	backendPvcLabel := fmt.Sprintf("%s=%s", mayav1alpha1.CASTypeKey, "nfs-kernel")
	pvcList, err := client.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: backendPvcLabel})
	if err != nil {
		klog.Errorf("Failed to list PVC, err=%s", err)
		return err
	}

	for _, pvc := range pvcList.Items {
		pvcExists, err := nfsPvcExists(client, pvc)
		if err != nil {
			// failed to check NFS PVC existence,
			// will check in next retry
			klog.Errorf("Failed to check NFS PVC for backendPVC=%s/%s, err=%v", ns, pvc.Name, err)
			continue
		}

		if pvcExists {
			// NFS PVC exists for backend PVC
			continue
		}

		// check if NFS PV exists for this PVC or not
		nfsPvName := ""
		fmt.Sscanf(pvc.Name, "nfs-%s", &nfsPvName)
		if nfsPvName == "" {
			continue
		}

		if pvTracker.Inprogress(nfsPvName) {
			// provisioner is processing request for this PV
			continue
		}

		pvExists, err := pvExists(client, nfsPvName)
		if err != nil {
			// failed to check pv existence, will check in next retry
			klog.Errorf("Failed to check NFS PV for backendPVC=%s/%s, err=%v", ns, pvc.Name, err)
			continue
		}

		if pvExists {
			// Relevant NFS PV exists for backend PVC
			continue
		}

		// perform cleanup for stale NFS resource for this backend PVC
		err = deleteBackendStaleResources(client, pvc.Namespace, nfsPvName)
		if err != nil {
			klog.Errorf("Failed to delete NFS resources for backendPVC=%s/%s, err=%v", ns, pvc.Name, err)
		}
	}

	return nil
}

func deleteBackendStaleResources(client kubernetes.Interface, nfsServerNs, nfsPvName string) error {
	klog.Infof("Deleting stale resources for PV=%s", nfsPvName)

	p := &Provisioner{
		kubeClient:      client,
		serverNamespace: nfsServerNs,
	}

	nfsServerOpts := &KernelNFSServerOptions{
		pvName: nfsPvName,
		ctx:    context.TODO(),
	}

	return p.deleteNFSServer(nfsServerOpts)
}

func nfsPvcExists(client kubernetes.Interface, backendPvcObj corev1.PersistentVolumeClaim) (bool, error) {
	nfsPvcName, nameExists := backendPvcObj.Labels[nfsPvcNameLabelKey]
	nfsPvcNs, nsExists := backendPvcObj.Labels[nfsPvcNsLabelKey]
	nfsPvcUID, uidExists := backendPvcObj.Labels[nfsPvcUIDLabelKey]

	if !nameExists || !nsExists || !uidExists {
		return false, errors.New("backend PVC doesn't have sufficient information of nfs pvc")
	}

	pvcObj, err := client.CoreV1().PersistentVolumeClaims(nfsPvcNs).Get(context.TODO(), nfsPvcName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			// couldn't get the nfs pvc information due to network error or
			// we don't have permission to fetch pvc from user namespace
			return false, err
		}
		return false, nil
	}

	if nfsPvcUID != string(pvcObj.UID) {
		klog.Infof("different UID=%s actual=%s", nfsPvcUID, string(pvcObj.UID))
		// pvc is having different UID than nfs PVC, so
		// original nfs pvc is deleted
		return false, nil
	}

	return true, nil
}

func pvExists(client kubernetes.Interface, pvName string) (bool, error) {
	_, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}
