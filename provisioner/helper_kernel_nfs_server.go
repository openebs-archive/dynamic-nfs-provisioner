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

package provisioner

import (
	errors "github.com/pkg/errors"
	"k8s.io/klog"

	deployment "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	persistentvolumeclaim "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	service "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/service"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pvcStorageClassAnnotation = "nfs.openebs.io/persistentvolumeclaim"
	pvStorageClassAnnotation  = "nfs.openebs.io/persistentvolume"

	// NFSPVFinalizer represents finalizer string used by NFSPV
	NFSPVFinalizer = "nfs.openebs.io/finalizer"

	//NFSServerPort set the NFS Server Port
	NFSServerPort = 2049

	//RPCBindPort set the RPC Bind Port
	RPCBindPort = 111
)

var (
	//WaitForNFSServerRetries specifies the number of times provisioner
	// should wait and check if the NFS server is initialized.
	//The duration is the value specified here multiplied by 5
	WaitForNFSServerRetries = 12
)

// KernelNFSServerOptions contains the options that
// will launch Kernel NFS Server using the provided storage
// class
type KernelNFSServerOptions struct {
	provisionerNS       string
	pvName              string
	capacity            string
	backendStorageClass string
	pvcName             string
	serviceName         string
	deploymentName      string
}

// validate checks that the required fields to create NFS Server
// are available
func (nfsServerOpts *KernelNFSServerOptions) validate() error {
	return nil
}

// createBackendPVC creates a new exports PVC for a given NFS PVC
func (p *Provisioner) createBackendPVC(nfsServerOpts *KernelNFSServerOptions) error {
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	pvcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if PVC(%v) for NFS storage was already created.", pvcName)

	//Check if the PVC is already created. This can happen
	//if the previous reconciliation of PVC-PV, resulted in
	//creating a PVC, but was not yet available for 60+ seconds
	_, err := persistentvolumeclaim.NewKubeClient().
		WithNamespace(p.namespace).
		Get(pvcName, metav1.GetOptions{})

	if err == nil {
		nfsServerOpts.pvcName = pvcName
		klog.Infof("Volume %v has been initialized with PVC:%v", nfsServerOpts.pvName, pvcName)
		return nil
	}

	//TODO
	// Create PVC using the provided capacity and SC details
	pvcObjBuilder := persistentvolumeclaim.NewBuilder().
		WithNamespace(p.namespace).
		WithName(pvcName).
		WithLabels(nfsServerOpts.getLabels()).
		WithCapacity(nfsServerOpts.capacity).
		WithAccessModeRWO().
		WithStorageClass(nfsServerOpts.backendStorageClass)

	pvcObj, err := pvcObjBuilder.Build()

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "unable to build PVC")
	}

	_, err = persistentvolumeclaim.NewKubeClient().
		WithNamespace(p.namespace).
		Create(pvcObj)

	if err != nil {
		//TODO : Need to relook at this error
		//If the error is about PVC being already present, then return nil
		return errors.Wrapf(err, "failed to create PVC{%v}", pvcName)
	}

	nfsServerOpts.pvcName = pvcName

	return nil
}

// deleteBackendPVC deletes the NFS Server Backend PVC for a given NFS PVC
func (p *Provisioner) deleteBackendPVC(nfsServerOpts *KernelNFSServerOptions) error {
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	pvcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if PVC(%v) for NFS storage exists.", pvcName)

	//Check if the PVC still exists. It could have been removed
	// or never created due to a provisioning create failure.
	_, err := persistentvolumeclaim.NewKubeClient().
		WithNamespace(p.namespace).
		Get(pvcName, metav1.GetOptions{})

	if err == nil {
		nfsServerOpts.pvcName = pvcName
		klog.Infof("Volume %v has been initialized with PVC:%v. Initiating delete...", nfsServerOpts.pvName, pvcName)
	} else {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete PVC
	err = persistentvolumeclaim.NewKubeClient().
		WithNamespace(p.namespace).
		Delete(pvcName, &metav1.DeleteOptions{})

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Errorf("unable to delete PVC %v associated with PV:%v", nfsServerOpts.pvName, pvcName)
	}
	return nil
}

// createDeployment creates a new NFS Server Deployment for a given NFS PVC
func (p *Provisioner) createDeployment(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Creating Deployment")
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	deployName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if Deployment(%v) for NFS storage was already created.", deployName)

	//Check if the Deployment is already created. This can happen
	//if the previous reconciliation of PVC-PV, resulted in
	//creating a Deployment, but was not yet available for 60+ seconds
	_, err := deployment.NewKubeClient().
		WithNamespace(p.namespace).
		Get(deployName)

	if err == nil {
		nfsServerOpts.deploymentName = deployName
		klog.Infof("Volume %v has been initialized with Deployment:%v", nfsServerOpts.pvName, deployName)
		return nil
	}

	nfsDeployLabelSelector := map[string]string{
		"openebs.io/nfs-server": deployName,
	}

	//TODO
	secContext := true

	// Create Deployment for NFS Server and mount the exports PVC.
	deployObjBuilder := deployment.NewBuilder().
		WithName(deployName).
		WithNamespace(p.namespace).
		WithLabelsNew(nfsDeployLabelSelector).
		WithSelectorMatchLabelsNew(nfsDeployLabelSelector).
		WithStrategyTypeRecreate().
		WithPodTemplateSpecBuilder(
			pts.NewBuilder().
				WithLabelsNew(nfsDeployLabelSelector).
				WithContainerBuildersNew(
					container.NewBuilder().
						WithName("nfs-server").
						WithImage("itsthenetwork/nfs-server-alpine").
						WithEnvsNew(
							[]corev1.EnvVar{
								{
									Name:  "SHARED_DIRECTORY",
									Value: "/nfsshare",
								},
							},
						).
						WithPortsNew(
							[]corev1.ContainerPort{
								{
									Name:          "nfs",
									ContainerPort: NFSServerPort,
								},
								{
									Name:          "rpcbind",
									ContainerPort: RPCBindPort,
								},
							},
						).
						WithPrivilegedSecurityContext(&secContext).
						WithVolumeMountsNew(
							[]corev1.VolumeMount{
								{
									Name:      "exports-dir",
									MountPath: "/nfsshare",
								},
							},
						),
				).
				WithVolumeBuilders(
					volume.NewBuilder().
						WithName("exports-dir").
						WithPVCSource(nfsServerOpts.pvcName),
				),
		)

	deployObj, err := deployObjBuilder.Build()

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "unable to build Deployment")
	}

	_, err = deployment.NewKubeClient().
		WithNamespace(p.namespace).
		Create(deployObj)

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "failed to create Deployment{%v}", deployName)
	}

	nfsServerOpts.deploymentName = deployName

	return nil
}

// deleteDeployment deletes the NFS Server Deployment for a given NFS PVC
func (p *Provisioner) deleteDeployment(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Deleting Deployment")
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	deployName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if Deployment(%v) for NFS storage exists.", deployName)

	//Check if the Deploy still exists. It could have been removed
	// or never created due to a provisioning create failure.
	_, err := deployment.NewKubeClient().
		WithNamespace(p.namespace).
		Get(deployName)

	if err == nil {
		nfsServerOpts.deploymentName = deployName
		klog.Infof("Volume %v has been initialized with Deployment:%v. Initiating delete...", nfsServerOpts.pvName, deployName)
	} else {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete PVC
	err = deployment.NewKubeClient().
		WithNamespace(p.namespace).
		Delete(deployName, &metav1.DeleteOptions{})

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Errorf("unable to delete deployment %v associated with PV:%v", nfsServerOpts.pvName, deployName)
	}

	return nil
}

// createService creates a new NFS Server Service for a given NFS PVC
func (p *Provisioner) createService(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Creating Service")
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	svcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if Service(%v) for NFS storage was already created.", svcName)

	//Check if the Service is already created. This can happen
	//if the previous reconciliation of PVC-PV, resulted in
	//creating a Service, but was not yet available for 60+ seconds
	_, err := service.NewKubeClient().
		WithNamespace(p.namespace).
		Get(svcName, metav1.GetOptions{})

	if err == nil {
		nfsServerOpts.serviceName = svcName
		klog.Infof("Volume %v has been initialized with Service:%v", nfsServerOpts.pvName, svcName)
		return nil
	}

	nfsDeployLabelSelector := map[string]string{
		"openebs.io/nfs-server": nfsServerOpts.deploymentName,
	}

	//TODO
	// Create Service
	svcObjBuilder := service.NewBuilder().
		WithNamespace(p.namespace).
		WithName(svcName).
		WithPorts(
			[]corev1.ServicePort{
				{
					Name: "nfs",
					Port: NFSServerPort,
				},
				{
					Name: "rpcbind",
					Port: RPCBindPort,
				},
			},
		).
		WithSelectorsNew(nfsDeployLabelSelector)

	svcObj, err := svcObjBuilder.Build()

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "unable to build Service")
	}

	_, err = service.NewKubeClient().
		WithNamespace(p.namespace).
		Create(svcObj)

	if err != nil {
		//TODO : Need to relook at this error
		//If the error is about PVC being already present, then return nil
		return errors.Wrapf(err, "failed to create Service{%v}", svcName)
	}

	nfsServerOpts.serviceName = svcName

	return nil
}

// deleteService deletes the NFS Server Service for a given NFS PVC
func (p *Provisioner) deleteService(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Deleting Service")
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	svcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if Service(%v) for NFS storage exists.", svcName)

	//Check if the Serivce still exists. It could have been removed
	// or never created due to a provisioning create failure.
	_, err := service.NewKubeClient().
		WithNamespace(p.namespace).
		Get(svcName, metav1.GetOptions{})

	if err == nil {
		nfsServerOpts.serviceName = svcName
		klog.Infof("Volume %v has been initialized with Service:%v. Initiating delete...", nfsServerOpts.pvName, svcName)
	} else {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete Service
	err = service.NewKubeClient().
		WithNamespace(p.namespace).
		Delete(svcName, &metav1.DeleteOptions{})

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Errorf("unable to delete Service %v associated with PV:%v", nfsServerOpts.pvName, svcName)
	}

	return nil
}

// getNFSServerAddress fetches the NFS Server Cluster IP associated with this PV
// or creates one.
func (p *Provisioner) getNFSServerAddress(nfsServerOpts *KernelNFSServerOptions) (string, error) {
	klog.V(4).Infof("Getting NFS Service Cluster IP")

	// Check if the NFS Service has been created (which is the last step
	// If not create NFS Service
	err := p.createNFSServer(nfsServerOpts)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create NFS Server for PVC{%v}", nfsServerOpts.pvName)
	}

	//Get the NFS Service to extract Cluster IP
	if p.useClusterIP {
		//nfsService := nil
		nfsService, err := service.NewKubeClient().
			WithNamespace(p.namespace).
			Get(nfsServerOpts.serviceName, metav1.GetOptions{})
		if err != nil || nfsService == nil {
			return "", errors.Wrapf(err, "failed to get NFS Service for PVC{%v}", nfsServerOpts.pvcName)
		}
		return nfsService.Spec.ClusterIP, nil
	}

	// Return the cluster local nfs service ip
	// <service-name>.<namespace>.svc.cluster.local
	return nfsServerOpts.serviceName + "." + p.namespace + ".svc.cluster.local", nil
}

// createNFSServer creates the NFS Server deployment and related
// objects created for the given PV
func (p *Provisioner) createNFSServer(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Create NFS Server")
	// Create PVC, Deployment and Service
	err := p.createBackendPVC(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize NFS Storage PVC for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	err = p.createDeployment(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize NFS Storage Deployment for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	err = p.createService(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize NFS Storage Service for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	//TODO
	// Add finalizers once the objects have been setup
	// Use the service to setup or return PV details
	return nil
}

// deleteNFSServer deletes the NFS Server deployment and related
// objects created for the given PV
func (p *Provisioner) deleteNFSServer(nfsServerOpts *KernelNFSServerOptions) error {
	klog.V(4).Infof("Delete NFS Server")

	err := p.deleteService(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to delete NFS Storage Service for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	err = p.deleteDeployment(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to delete NFS Storage Deployment for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	err = p.deleteBackendPVC(nfsServerOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to delete NFS Storage PVC for RWX PVC{%v}", nfsServerOpts.pvName)
	}

	return nil
}

func (nfsServerOpts *KernelNFSServerOptions) getLabels() map[string]string {
	return map[string]string{
		"persistent-volume":   nfsServerOpts.pvName,
		"openebs.io/cas-type": "nfs-kernel",
	}
}
