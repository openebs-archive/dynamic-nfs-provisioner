/*
Copyright 2019 The OpenEBS Authors.

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
	"github.com/openebs/maya/pkg/alertlog"
	"github.com/pkg/errors"
	"k8s.io/klog"

	mPV "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolume"
	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	v1 "k8s.io/api/core/v1"
	pvController "sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

// ProvisionKernalNFSServer is invoked by the Provisioner to create a NFS
//  with kernel NFS server
func (p *Provisioner) ProvisionKernalNFSServer(opts pvController.ProvisionOptions, volumeConfig *VolumeConfig) (*v1.PersistentVolume, error) {
	var leaseTime, graceTime int
	var leaseErr, graceErr error

	pvc := opts.PVC
	name := opts.PVName
	capacity := opts.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]

	leaseTime, leaseErr = volumeConfig.GetNFSServerLeaseTime()
	graceTime, graceErr = volumeConfig.GetNFServerGraceTime()
	if leaseErr != nil || graceErr != nil {
		klog.Errorf("Error parsing lease/grace time, leaseError=%s graceError=%s", leaseErr, graceErr)
		alertlog.Logger.Errorw("",
			"eventcode", "nfs.pv.provision.failure",
			"msg", "Failed to provision NFS PV",
			"rname", opts.PVName,
			"reason", "Parsing failed for lease/grace time",
			"storagetype", "nfs-kernel",
		)
	}
	fsGID, err := volumeConfig.GetFSGroupID()
	if err != nil {
		klog.Errorf("Error parsing fsgid error: %s", err.Error())
		return nil, err
	}

	//Extract the details to create a NFS Server
	nfsServerOpts := &KernelNFSServerOptions{
		pvName:                name,
		provisionerNS:         p.namespace,
		capacity:              capacity.String(),
		backendStorageClass:   volumeConfig.GetBackendStorageClassFromConfig(),
		nfsServerCustomConfig: volumeConfig.GetCustomNFSServerConfig(),
		leaseTime:             leaseTime,
		graceTime:             graceTime,
		fsGroup:               fsGID,
	}

	nfsService, err := p.getNFSServerAddress(nfsServerOpts)

	if err != nil {
		klog.Infof("Initialize volume %v failed: %v", name, err)
		alertlog.Logger.Errorw("",
			"eventcode", "nfs.pv.provision.failure",
			"msg", "Failed to provision NFS PV",
			"rname", opts.PVName,
			"reason", "NFS service initialization failed",
			"storagetype", "nfs-kernel",
		)
		return nil, err
	}

	klog.Infof("Creating nfs volume %v pointing at %v", name, nfsService)

	// TODO initialize the Labels and annotations
	// Use annotations to specify the context using which the PV was created.
	// Add the server type as kernel
	//volAnnotations := make(map[string]string)
	//volAnnotations[bdcStorageClassAnnotation] = blkDevOpts.bdcName
	//fstype := casVolume.Spec.FSType

	labels := make(map[string]string)
	labels[string(mconfig.CASTypeKey)] = "nfs-kernel"
	//labels[string(v1alpha1.StorageClassKey)] = *className

	//TODO Change the following to a builder pattern
	// Add NFS Server Options
	pvObjBuilder := mPV.NewBuilder().
		WithName(name).
		WithLabels(labels).
		//WithAnnotations(volAnnotations).
		WithReclaimPolicy(*opts.StorageClass.ReclaimPolicy).
		WithAccessModes(pvc.Spec.AccessModes).
		WithCapacityQty(pvc.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]).
		WithMountOptions(opts.StorageClass.MountOptions).
		WithNFS(nfsService, "/", false)

	//Note: The nfs server is launched by the nfs-server-alpine.
	//When "/" is replaced with "/nfsshare", the mount fails.
	//
	//Ref: https://github.com/sjiveson/nfs-server-alpine
	//Due to the fsid=0 parameter set in the /etc/exports file,
	//there's no need to specify the folder name when mounting from a client.
	//For example, this works fine even though the folder being mounted and
	//shared is /nfsshare

	//Build the pvObject
	pvObj, err := pvObjBuilder.Build()

	if err != nil {
		alertlog.Logger.Errorw("",
			"eventcode", "nfs.pv.provision.failure",
			"msg", "Failed to provision NFS PV",
			"rname", opts.PVName,
			"reason", "Building volume failed",
			"storagetype", "nfs-kernel",
		)
		return nil, err
	}
	alertlog.Logger.Infow("",
		"eventcode", "nfs.pv.provision.success",
		"msg", "Successfully provisioned NFS PV",
		"rname", opts.PVName,
		"storagetype", "nfs-kernel",
	)
	return pvObj, nil
}

// DeleteKernalNFSServer is invoked by the PVC controller to perform clean-up
//  activities before deleteing the PV object. If reclaim policy is
//  set to not-retain, then this function will delete the associated BDC
func (p *Provisioner) DeleteKernalNFSServer(pv *v1.PersistentVolume) (err error) {
	defer func() {
		err = errors.Wrapf(err, "failed to delete volume %v", pv.Name)
	}()

	//Extract the details to delete NFS Server
	nfsServerOpts := &KernelNFSServerOptions{
		pvName: pv.Name,
	}

	return p.deleteNFSServer(nfsServerOpts)
}
