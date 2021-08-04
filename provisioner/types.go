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
	"time"

	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
)

//Provisioner struct has the configuration and utilities required
// across the different work-flows.
type Provisioner struct {
	stopCh chan struct{}

	kubeClient clientset.Interface

	// namespace in which provisioner is running
	namespace string

	// serverNamespace in which nfs server deployments gets created
	// can be set through env variable NFS_SERVER_NAMESPACE
	// default value is Provisioner.namespace
	serverNamespace string

	// defaultConfig is the default configurations
	// provided from ENV or Code
	defaultConfig []mconfig.Config

	// getVolumeConfig is a reference to a function
	getVolumeConfig GetVolumeConfigFn

	//determine if clusterIP or clusterDNS should be used
	useClusterIP bool

	// k8sNodeLister hold cache information about nodes
	k8sNodeLister listerv1.NodeLister

	// nodeAffinity specifies requirements for scheduling NFS Server
	nodeAffinity NodeAffinity

	// markResourceForVolumeEvents to set required annotation/finalizer
	// on NFS resources to send events
	markResourceForVolumeEvents bool

	// backendPvcTimeout defines timeout for backend PVC Bound check
	backendPvcTimeout time.Duration
}

//VolumeConfig struct contains the merged configuration of the PVC
// and the associated SC. The configuration is derived from the
// annotation `cas.openebs.io/config`. The configuration will be
// in the following json format:
// {
//   Key1:{
//	enabled: true
//	value: "string value"
//   },
//   Key2:{
//	enabled: true
//	value: "string value"
//   },
// }
type VolumeConfig struct {
	pvName  string
	pvcName string
	scName  string
	options map[string]interface{}
}

// GetVolumeConfigFn allows to plugin a custom function
//  and makes it easy to unit test provisioner
type GetVolumeConfigFn func(pvName string, pvc *corev1.PersistentVolumeClaim) (*VolumeConfig, error)

// NodeAffinity represents group of node affinity scheduling
// rules that will be applied on NFS Server instance. If it is
// not configured then matches to no object i.e NFS Server can
// schedule on any node in a cluster. Configured values will be
// propogated to deployment.spec.template.spec.affinity.nodeAffinity.
//					requiredDuringSchedulingIgnoredDuringExecution
//
// Values are propagated via ENV(NodeAffinity) on NFS Provisioner.
// Example: Following can be various options to specify NodeAffinity rules
//
//		Config 1: Configure across zones and also storage should be available
//			Env Value: "kubernetes.io/hostName:[z1-host1,z2-host1,z3-host1],kubernetes.io/storage:[available]"
//
//  		Config 1 will be propogated as shown below on NFS-Server deployment
//  			nodeSelectorTerms:
//  			- matchExpressions:
//  			  - key: kubernetes.io/hostName
//  				operator: "In"
//  			    values:
//  			    - z1-host1
//  				- z2-host2
//  				- z3-host3
//  			  - key: kubernetes.io/storage
//  			    operator: "In"
//  				values:
//  				- available
//
//      Config2: Configure on storage nodes in zone1
//			Env Value: "kubernetes.io/storage:[],kubernetes.io/zone:[zone1]"
//
//  		Config2 will be propogated as shown below on NFS-Server deployment
//  			nodeSelectorTerms:
//  			- matchExpressions:
//  			  - key: kubernetes.io/storage
//  			    operator: "Exists"
//  			  - key: kubernetes.io/zone
//  				operator: "In"
//  			    values:
//  			    - zone1
//
//
//		Configi3: Configure on any storage node
//			Env Value: "kubernetes.io/storage:[]"
//
//  		Config3 will be propogated as below on NFS-Server deployment
//  			nodeSelectorTerms:
//  			- matchExpressions:
//  			  - key: kubernetes.io/storage
//  			    operator: "Exists"
//
//      Like shown above various combinations can be specified and before
//		provisioning configuration will be validated
//
// NOTE: All the comma separated specification will be ANDed
type NodeAffinity struct {
	// A list of node selector requirements by node's labels
	MatchExpressions []corev1.NodeSelectorRequirement
}
