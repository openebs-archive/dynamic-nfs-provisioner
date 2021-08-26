/*
Copyright 2020 The OpenEBS Authors.

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

/*
This file contains the volume creation and deletion handlers invoked by
the github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/controller.

The handler that are madatory to be implemented:

- Provision - is called by controller to perform custom validation on the PVC
  request and return a valid PV spec. The controller will create the PV object
  using the spec passed to it and bind it to the PVC.

- Delete - is called by controller to perform cleanup tasks on the PV before
  deleting it.

*/

package provisioner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/openebs/maya/pkg/alertlog"

	"github.com/pkg/errors"
	"k8s.io/klog"
	pvController "sigs.k8s.io/sig-storage-lib-external-provisioner/controller"

	"github.com/openebs/dynamic-nfs-provisioner/pkg/metrics"
	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	menv "github.com/openebs/maya/pkg/env/v1alpha1"
	analytics "github.com/openebs/maya/pkg/usage"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kubeinformers "k8s.io/client-go/informers"
	listersv1 "k8s.io/client-go/listers/core/v1"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	NodeAffinityRulesMismatchEvent = "No matching nodes found for given affinity rules"
)

// NewProvisioner will create a new Provisioner object and initialize
//  it with global information used across PV create and delete operations.
func NewProvisioner(stopCh chan struct{}, kubeClient *clientset.Clientset) (*Provisioner, error) {

	namespace := getOpenEBSNamespace()
	if len(strings.TrimSpace(namespace)) == 0 {
		return nil, fmt.Errorf("Cannot start Provisioner: failed to get namespace")
	}

	backendPvcTimeoutStr := getBackendPvcTimeout()
	backendPvcTimeoutVal, err := strconv.Atoi(backendPvcTimeoutStr)
	if err != nil || backendPvcTimeoutVal == 0 {
		klog.Warningf("Invalid backendPvcTimeout value=%s, using default value %d", backendPvcTimeoutStr, DefaultBackendPvcBoundTimeout)
		backendPvcTimeoutVal = DefaultBackendPvcBoundTimeout
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	k8sNodeInformer := kubeInformerFactory.Core().V1().Nodes().Informer()

	nfsServerNs := getNfsServerNamespace()

	pvTracker := NewProvisioningTracker()

	p := &Provisioner{
		stopCh: stopCh,

		kubeClient:      kubeClient,
		namespace:       namespace,
		serverNamespace: nfsServerNs,
		defaultConfig: []mconfig.Config{
			{
				Name:  KeyPVNFSServerType,
				Value: getDefaultNFSServerType(),
			},
		},
		useClusterIP:      menv.Truthy(ProvisionerNFSServerUseClusterIP),
		k8sNodeLister:     listersv1.NewNodeLister(k8sNodeInformer.GetIndexer()),
		nodeAffinity:      getNodeAffinityRules(),
		pvTracker:         pvTracker,
		backendPvcTimeout: time.Duration(backendPvcTimeoutVal) * time.Second,
	}
	p.getVolumeConfig = p.GetVolumeConfig

	// Running node informer will fetch node information from API Server
	// and maintain it in cache
	go k8sNodeInformer.Run(stopCh)

	// Running garbage collector to perform cleanup for stale NFS resources
	go RunGarbageCollector(kubeClient, pvTracker, nfsServerNs, stopCh)

	return p, nil
}

// SupportsBlock will be used by controller to determine if block mode is
//  supported by the host path provisioner.
func (p *Provisioner) SupportsBlock() bool {
	return false
}

// Provision is invoked by the PVC controller which expect the PV
//  to be provisioned and a valid PV spec returned.
func (p *Provisioner) Provision(opts pvController.ProvisionOptions) (*v1.PersistentVolume, error) {
	pvc := opts.PVC

	p.pvTracker.Add(opts.PVName)
	defer p.pvTracker.Delete(opts.PVName)

	for _, accessMode := range pvc.Spec.AccessModes {
		if accessMode != v1.ReadWriteMany {
			klog.Infof("Received PVC provision request for non-rwx mode %v", accessMode)
		}
	}

	name := opts.PVName

	// Create a new Config instance for the PV by merging the
	// default configuration with configuration provided
	// via PVC and the associated StorageClass
	pvCASConfig, err := p.getVolumeConfig(name, pvc)
	if err != nil {
		return nil, err
	}

	// Validate nodeAffinity rules for scheduling
	// There might be changes to node after deploying
	// NFS Provisioner
	err = p.validateNodeAffinityRules()
	if err != nil {
		return nil, err
	}

	nfsServerType := pvCASConfig.GetNFSServerTypeFromConfig()

	size := resource.Quantity{}
	reqMap := pvc.Spec.Resources.Requests
	if reqMap != nil {
		size = pvc.Spec.Resources.Requests["storage"]
	}

	sendEventOrIgnore(pvc.Name, name, size.String(), nfsServerType, analytics.VolumeProvision)

	if nfsServerType == "kernel" {
		pv, err := p.ProvisionKernalNFSServer(opts, pvCASConfig)
		if err != nil {
			metrics.PersistentVolumeCreateFailedTotal.WithLabelValues(metrics.ProvisionerRequestCreate).Inc()
			return nil, err
		}
		metrics.PersistentVolumeCreateTotal.WithLabelValues(metrics.ProvisionerRequestCreate).Inc()
		return pv, nil
	}

	alertlog.Logger.Errorw("",
		"eventcode", "nfs.pv.provision.failure",
		"msg", "Failed to provision NFS PV",
		"rname", opts.PVName,
		"reason", "NFSServerType not supported",
		"storagetype", nfsServerType,
	)
	return nil, fmt.Errorf("PV with NFS Server of type(%v) is not supported", nfsServerType)
}

// Delete is invoked by the PVC controller to perform clean-up
//  activities before deleteing the PV object. If reclaim policy is
//  set to not-retain, then this function will create a helper pod
//  to delete the host path from the node.
func (p *Provisioner) Delete(pv *v1.PersistentVolume) (err error) {
	p.pvTracker.Add(pv.Name)
	defer p.pvTracker.Delete(pv.Name)

	defer func() {
		err = errors.Wrapf(err, "failed to delete volume %v", pv.Name)
	}()
	//Initiate clean up only when reclaim policy is not retain.
	if pv.Spec.PersistentVolumeReclaimPolicy != v1.PersistentVolumeReclaimRetain {

		//TODO: extract the nfs server type from PV annotation
		nfsServerType := GetNFSServerTypeFromPV(pv)
		pvType := "nfs-" + nfsServerType

		size := resource.Quantity{}
		reqMap := pv.Spec.Capacity
		if reqMap != nil {
			size = pv.Spec.Capacity["storage"]
		}

		pvcName := ""
		if pv.Spec.ClaimRef != nil {
			pvcName = pv.Spec.ClaimRef.Name
		}
		sendEventOrIgnore(pvcName, pv.Name, size.String(), pvType, analytics.VolumeDeprovision)

		if nfsServerType == "kernel" {
			err = p.DeleteKernalNFSServer(pv)
		}

		if err == nil {
			metrics.PersistentVolumeDeleteTotal.WithLabelValues(metrics.ProvisionerRequestDelete).Inc()
			return nil
		}

		alertlog.Logger.Errorw("",
			"eventcode", "nfs.pv.delete.failure",
			"msg", "Failed to delete NFS PV",
			"rname", pv.Name,
			"reason", "failed to delete NFS Server",
			"storagetype", pvType,
		)
		metrics.PersistentVolumeDeleteFailedTotal.WithLabelValues(metrics.ProvisionerRequestDelete).Inc()
		return err
	}
	klog.Infof("Retained volume %v", pv.Name)
	alertlog.Logger.Infow("",
		"eventcode", "nfs.pv.delete.success",
		"msg", "Successfully deleted NFS PV",
		"rname", pv.Name,
	)
	return nil
}

// validateNodeAffinityRules will returns error if there are no
// node exist for given affinity rules
func (p *Provisioner) validateNodeAffinityRules() error {
	if len(p.nodeAffinity.MatchExpressions) == 0 {
		return nil
	}

	nodeSelector, err := v1helper.NodeSelectorRequirementsAsSelector(p.nodeAffinity.MatchExpressions)
	if err != nil {
		return err
	}

	nodeList, err := p.k8sNodeLister.List(nodeSelector)
	if err != nil {
		return err
	}

	if len(nodeList) == 0 {
		return errors.Errorf("%s (%s)", NodeAffinityRulesMismatchEvent, nodeSelector.String())
	}
	return nil
}

// sendEventOrIgnore sends anonymous nfs-pv provision/delete events
func sendEventOrIgnore(pvcName, pvName, capacity, stgType, method string) {
	if !menv.Truthy(menv.OpenEBSEnableAnalytics) {
		return
	}

	if method == analytics.VolumeProvision {
		stgType = "nfs-" + stgType
	}

	analytics.New().Build().ApplicationBuilder().
		SetVolumeType(stgType, method).
		SetDocumentTitle(pvName).
		SetCampaignName(pvcName).
		SetLabel(analytics.EventLabelCapacity).
		SetReplicaCount("", method).
		SetCategory(method).
		SetVolumeCapacity(capacity).Send()
}
