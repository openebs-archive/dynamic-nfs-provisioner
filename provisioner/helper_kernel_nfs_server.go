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
	"strconv"
	"time"

	errors "github.com/pkg/errors"
	"k8s.io/klog"

	deployment "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/apps/v1/deployment"
	container "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/container"
	persistentvolumeclaim "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/persistentvolumeclaim"
	pts "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/podtemplatespec"
	service "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/service"
	volume "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/api/core/v1/volume"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	pvcStorageClassAnnotation = "nfs.openebs.io/persistentvolumeclaim"
	pvStorageClassAnnotation  = "nfs.openebs.io/persistentvolume"

	// PVC Label key to store information about NFS PVC
	nfsPvcNameLabelKey = "nfs.openebs.io/nfs-pvc-name"
	nfsPvcUIDLabelKey  = "nfs.openebs.io/nfs-pvc-uid"
	nfsPvcNsLabelKey   = "nfs.openebs.io/nfs-pvc-namespace"

	// NFSPVFinalizer represents finalizer string used by NFSPV
	NFSPVFinalizer = "nfs.openebs.io/finalizer"

	//NFSServerPort set the NFS Server Port
	NFSServerPort = 2049

	//RPCBindPort set the RPC Bind Port
	RPCBindPort = 111

	// DefaultBackendPvcBoundTimeout defines the timeout for PVC Bound check.
	// set to 60 seconds
	DefaultBackendPvcBoundTimeout = 60
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
	provisionerNS         string
	pvName                string
	capacity              string
	pvcName               string
	pvcUID                string
	pvcNamespace          string
	backendStorageClass   string
	backendPvcName        string
	serviceName           string
	deploymentName        string
	nfsServerCustomConfig string

	// leaseTime defines the renewal period(in seconds) for client state
	// this should be in range from 10 to 3600 seconds
	leaseTime int

	// graceTime defines the recovery period(in seconds) to reclaim
	// the locks and state
	graceTime int

	// fsGID defines the filesystem group ID if set then nfs share
	// volume permissions will be updated by OR'ing with rw-rw----
	fsGroup *int64

	// resources defines the request & limits of NFS server
	// This will be populated from NFS StorageClass. If not
	// specified resource limits will not be applied on NFS
	// Server container
	resources *corev1.ResourceRequirements
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

	backendPvcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if PVC(%v) for NFS storage was already created.", backendPvcName)

	//Check if the PVC is already created. This can happen
	//if the previous reconciliation of PVC-PV, resulted in
	//creating a PVC, but was not yet available for 60+ seconds
	_, err := p.kubeClient.CoreV1().
		PersistentVolumeClaims(p.serverNamespace).
		Get(backendPvcName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to check existence of backend PVC {%s/%s}", p.serverNamespace, backendPvcName)
	} else if err == nil {
		nfsServerOpts.backendPvcName = backendPvcName
		klog.Infof("Volume %v has been initialized with PVC {%s/%s}", nfsServerOpts.pvName, p.serverNamespace, backendPvcName)
		return nil
	}

	pvcLabel := nfsServerOpts.getLabels()
	pvcLabel[nfsPvcNameLabelKey] = nfsServerOpts.pvcName
	pvcLabel[nfsPvcUIDLabelKey] = nfsServerOpts.pvcUID
	pvcLabel[nfsPvcNsLabelKey] = nfsServerOpts.pvcNamespace

	// Create PVC using the provided capacity and SC details
	pvcObjBuilder := persistentvolumeclaim.NewBuilder().
		WithNamespace(p.serverNamespace).
		WithName(backendPvcName).
		WithLabels(pvcLabel).
		WithCapacity(nfsServerOpts.capacity).
		WithAccessModeRWO().
		WithStorageClass(nfsServerOpts.backendStorageClass)

	pvcObj, err := pvcObjBuilder.Build()

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "unable to build PVC {%s/%s}", pvcObj.Namespace, pvcObj.Name)
	}

	_, err = p.kubeClient.CoreV1().
		PersistentVolumeClaims(p.serverNamespace).
		Create(pvcObj)
	if err != nil {
		return errors.Wrapf(err, "failed to create PVC {%s/%s}", p.serverNamespace, backendPvcName)
	}

	nfsServerOpts.backendPvcName = backendPvcName

	return nil
}

// deleteBackendPVC deletes the NFS Server Backend PVC for a given NFS PVC
func (p *Provisioner) deleteBackendPVC(nfsServerOpts *KernelNFSServerOptions) error {
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	backendPvcName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if PVC {%s/%s} for NFS storage exists.", p.serverNamespace, backendPvcName)

	//Check if the PVC still exists. It could have been removed
	// or never created due to a provisioning create failure.
	_, err := p.kubeClient.CoreV1().
		PersistentVolumeClaims(p.serverNamespace).
		Get(backendPvcName, metav1.GetOptions{})
	if err == nil {
		nfsServerOpts.backendPvcName = backendPvcName
		klog.Infof("Volume %v has been initialized with PVC {%s/%s} Initiating delete...", nfsServerOpts.pvName, p.serverNamespace, backendPvcName)
	} else if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete PVC
	err = p.kubeClient.CoreV1().
		PersistentVolumeClaims(p.serverNamespace).
		Delete(backendPvcName, &metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete backend PVC {%s/%s} associated with PV %v", p.serverNamespace, backendPvcName, nfsServerOpts.pvName)
	}
	return nil
}

// createDeployment creates a new NFS Server Deployment for a given NFS PVC
func (p *Provisioner) createDeployment(nfsServerOpts *KernelNFSServerOptions) error {
	var resourceRequirements corev1.ResourceRequirements
	klog.V(4).Infof("Creating Deployment")
	if err := nfsServerOpts.validate(); err != nil {
		return err
	}

	deployName := "nfs-" + nfsServerOpts.pvName
	klog.V(4).Infof("Verifying if Deployment(%v) for NFS storage was already created.", deployName)

	//Check if the Deployment is already created. This can happen
	//if the previous reconciliation of PVC-PV, resulted in
	//creating a Deployment, but was not yet available for 60+ seconds
	_, err := p.kubeClient.AppsV1().
		Deployments(p.serverNamespace).
		Get(deployName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to check existence of NFS server deployment {%s/%s}", p.serverNamespace, deployName)
	}
	if err == nil {
		nfsServerOpts.deploymentName = deployName
		klog.Infof("Volume %v has been initialized with Deployment {%s/%s}", nfsServerOpts.pvName, p.serverNamespace, deployName)
		return nil
	}

	nfsDeployLabelSelector := map[string]string{
		"openebs.io/nfs-server": deployName,
	}
	if nfsServerOpts.resources != nil {
		resourceRequirements = *nfsServerOpts.resources
	}

	//TODO
	secContext := true

	// Create Deployment for NFS Server and mount the exports PVC.
	deployObjBuilder := deployment.NewBuilder().
		WithName(deployName).
		WithNamespace(p.serverNamespace).
		WithLabelsNew(nfsDeployLabelSelector).
		WithSelectorMatchLabelsNew(nfsDeployLabelSelector).
		WithStrategyTypeRecreate().
		WithPodTemplateSpecBuilder(
			pts.NewBuilder().
				WithLabelsNew(nfsDeployLabelSelector).
				WithSecurityContext(&corev1.PodSecurityContext{
					FSGroup: nfsServerOpts.fsGroup,
				}).
				WithNodeAffinityMatchExpressions(p.nodeAffinity.MatchExpressions).
				WithContainerBuildersNew(
					container.NewBuilder().
						WithName("nfs-server").
						WithImage(getNFSServerImage()).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithEnvsNew(
							[]corev1.EnvVar{
								{
									Name:  "SHARED_DIRECTORY",
									Value: "/nfsshare",
								},
								{
									Name:  "CUSTOM_EXPORTS_CONFIG",
									Value: nfsServerOpts.nfsServerCustomConfig,
								},
								{
									Name:  "NFS_LEASE_TIME",
									Value: strconv.Itoa(nfsServerOpts.leaseTime),
								},
								{
									Name:  "NFS_GRACE_TIME",
									Value: strconv.Itoa(nfsServerOpts.graceTime),
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
						).
						WithResources(&resourceRequirements),
				).
				WithVolumeBuilders(
					volume.NewBuilder().
						WithName("exports-dir").
						WithPVCSource(nfsServerOpts.backendPvcName),
				),
		)

	deployObj, err := deployObjBuilder.Build()

	if err != nil {
		//TODO : Need to relook at this error
		return errors.Wrapf(err, "unable to build Deployment")
	}

	_, err = p.kubeClient.AppsV1().
		Deployments(p.serverNamespace).
		Create(deployObj)
	if err != nil {
		return errors.Wrapf(err, "failed to create NFS server deployment {%s/%s}", p.serverNamespace, deployName)
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
	_, err := p.kubeClient.AppsV1().
		Deployments(p.serverNamespace).
		Get(deployName, metav1.GetOptions{})
	if err == nil {
		nfsServerOpts.deploymentName = deployName
		klog.Infof("Volume %v has been initialized with Deployment:%v. Initiating delete...", nfsServerOpts.pvName, deployName)
	} else if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete NFS Server Deployment
	err = p.kubeClient.AppsV1().
		Deployments(p.serverNamespace).
		Delete(deployName, &metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete NFS server deployment {%s/%s} associated with PV %s", p.serverNamespace, deployName, nfsServerOpts.pvName)
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
	_, err := p.kubeClient.CoreV1().
		Services(p.serverNamespace).
		Get(svcName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to check existence of NFS service {%s/%s} of volume %s", p.serverNamespace, svcName, nfsServerOpts.pvName)
	} else if err == nil {
		nfsServerOpts.serviceName = svcName
		klog.Infof("Volume %v has been initialized with service {%s/%s}", nfsServerOpts.pvName, p.serverNamespace, svcName)
		return nil
	}

	nfsDeployLabelSelector := map[string]string{
		"openebs.io/nfs-server": nfsServerOpts.deploymentName,
	}

	//TODO
	// Create Service
	svcObjBuilder := service.NewBuilder().
		WithNamespace(p.serverNamespace).
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

	_, err = p.kubeClient.CoreV1().
		Services(p.serverNamespace).
		Create(svcObj)
	if err != nil {
		//TODO : Need to relook at this error
		//If the error is about PVC being already present, then return nil
		return errors.Wrapf(err, "failed to create NFS service {%s/%s} of volume %s", p.serverNamespace, svcName, nfsServerOpts.pvName)
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

	//Check if the Service still exists. It could have been removed
	// or never created due to a provisioning create failure.
	_, err := p.kubeClient.CoreV1().
		Services(p.serverNamespace).
		Get(svcName, metav1.GetOptions{})
	if err == nil {
		nfsServerOpts.serviceName = svcName
		klog.Infof("Volume %s has been initialized with Service {%s/%s}. Initiating delete...", nfsServerOpts.pvName, p.serverNamespace, svcName)
	} else if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	//TODO
	// remove finalizer

	// Delete Service
	err = p.kubeClient.CoreV1().
		Services(p.serverNamespace).
		Delete(svcName, &metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete NFS service %s/%s associated with PV:%s", p.serverNamespace, svcName, nfsServerOpts.pvName)
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
		return "", errors.Wrapf(err, "failed to deploy NFS Server")
	}

	//Get the NFS Service to extract Cluster IP
	if p.useClusterIP {
		//nfsService := nil
		nfsService, err := p.kubeClient.CoreV1().
			Services(p.serverNamespace).
			Get(nfsServerOpts.serviceName, metav1.GetOptions{})
		if err != nil || nfsService == nil {
			return "", errors.Wrapf(err, "failed to get NFS Service for PVC{%v}", nfsServerOpts.backendPvcName)
		}
		return nfsService.Spec.ClusterIP, nil
	}

	// Return the cluster local nfs service ip
	// <service-name>.<namespace>.svc.cluster.local
	return nfsServerOpts.serviceName + "." + p.serverNamespace + ".svc.cluster.local", nil
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

	err = waitForPvcBound(p.kubeClient, p.serverNamespace, "nfs-"+nfsServerOpts.pvName, p.backendPvcTimeout)
	if err != nil {
		return err
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

// waitForPvcBound wait for PVC to bound for timeout period
func waitForPvcBound(client kubernetes.Interface, namespace, name string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	timeoutCh := timer.C

	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-timeoutCh:
			return errors.Errorf("timed out waiting for PVC{%s/%s} to bound", namespace, name)

		case <-tick.C:
			obj, err := client.CoreV1().
				PersistentVolumeClaims(namespace).
				Get(name, metav1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to get pvc{%s/%s}", namespace, name)
			}

			if obj.Status.Phase == corev1.ClaimBound {
				return nil
			}
		}
	}
}
