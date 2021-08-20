/*
Copyright 2019 The OpenEBS Authors

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
	menv "github.com/openebs/maya/pkg/env/v1alpha1"
)

//This file defines the environment variable names that are specific
// to this provisioner. In addition to the variables defined in this file,
// provisioner also uses the following:
//   OPENEBS_NAMESPACE
//   OPENEBS_SERVICE_ACCOUNT
//   OPENEBS_IO_K8S_MASTER
//   OPENEBS_IO_KUBE_CONFIG

const (
	// ProvisionerNFSServerType is the environment variable that
	// allows user to specify the default NFS Server type to be used.
	ProvisionerNFSServerType menv.ENVKey = "OPENEBS_IO_NFS_SERVER_TYPE"

	// ProvisionerExportsSC is the environment variable that provides the
	// default storage class to be used for exports PVC mount used by NFS Server.
	ProvisionerExportsSC menv.ENVKey = "OPENEBS_IO_EXPORTS_SC"

	// ProvisionerNFSServerUseClusterIP is the environment variable that
	// allows user to specify if ClusterIP should be used in NFS K8s Service
	ProvisionerNFSServerUseClusterIP menv.ENVKey = "OPENEBS_IO_NFS_SERVER_USE_CLUSTERIP"

	// NFSServerImageKey is the environment variable that
	// store the container image name to be used for nfs-server deployment
	//
	// Note: If image name is not mentioned then provisioner.ProvisionerNFSServerImage
	//
	NFSServerImageKey menv.ENVKey = "OPENEBS_IO_NFS_SERVER_IMG"

	// NFSServerNamespace defines the namespace for nfs server objects
	// Default value is menv.OpenEBSNamespace(operator namespace)
	NFSServerNamespace menv.ENVKey = "OPENEBS_IO_NFS_SERVER_NS"

	// NodeAffinityKey holds the env name representing Node affinity rules
	NodeAffinityKey menv.ENVKey = "OPENEBS_IO_NFS_SERVER_NODE_AFFINITY"

	// NFSHookConfigMapName defines env variable name to hold hook configmap name
	NFSHookConfigMapName menv.ENVKey = "OPENEBS_IO_NFS_HOOK_CONFIGMAP"

	// NFSBackendPvcTimeout defines env name to store BackendPvcBoundTimeout value
	NFSBackendPvcTimeout menv.ENVKey = "OPENEBS_IO_NFS_SERVER_BACKEND_PVC_TIMEOUT"
)

var (
	defaultNFSServerType = "kernel"
	defaultExportsSC     = ""

	// NFSServerDefaultImage specifies the image name to be used in
	// nfs server deployment. If image name is mentioned as a env variable
	// provisioner.NFSServerImageKey then value from env variable will be used
	NFSServerDefaultImage string
)

func getOpenEBSNamespace() string {
	return menv.Get(menv.OpenEBSNamespace)
}

// getNfsServerNamespace return namespace for nfs-server
func getNfsServerNamespace() string {
	return menv.GetOrDefault(NFSServerNamespace, menv.Get(menv.OpenEBSNamespace))
}

func getDefaultExportsSC() string {
	return menv.GetOrDefault(ProvisionerExportsSC, string(defaultExportsSC))
}

func getDefaultNFSServerType() string {
	return menv.GetOrDefault(ProvisionerNFSServerType, string(defaultNFSServerType))
}

func getOpenEBSServiceAccountName() string {
	return menv.Get(menv.OpenEBSServiceAccount)
}

func getNFSServerImage() string {
	return menv.GetOrDefault(NFSServerImageKey, string(NFSServerDefaultImage))
}

func getNfsServerNodeAffinity() string {
	return menv.Get(NodeAffinityKey)
}

func getHookConfigMapName() string {
	return menv.Get(NFSHookConfigMapName)
}

func getBackendPvcTimeout() string {
	return menv.Get(NFSBackendPvcTimeout)
}
