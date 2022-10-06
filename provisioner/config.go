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
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/ghodss/yaml"
	nfshook "github.com/openebs/dynamic-nfs-provisioner/pkg/hook"
	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	cast "github.com/openebs/maya/pkg/castemplate/v1alpha1"
	"github.com/openebs/maya/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	//KeyPVNFSServerType defines if the NFS PV should be launched
	// using kernel or ganesha
	KeyPVNFSServerType = "NFSServerType"

	//KeyPVBackendStorageClass defines default provisioner to be used
	// to create the data(export) directory for NFS server
	KeyPVBackendStorageClass = "BackendStorageClass"

	//CustomServerConfig defines the server configuration to use,
	// if it is set. Otherwise, use the default NFS server configuration.
	CustomServerConfig = "CustomServerConfig"

	// LeaseTime defines the renewal period(in seconds) for client state
	// if not set then default value(90s) will be used
	LeaseTime        = "LeaseTime"
	DefaultLeaseTime = 90

	// GraceTime defines the recovery period(in seconds) to reclaim locks
	// If it is not set then default value(90s) will be used
	GraceTime        = "GraceTime"
	DefaultGraceTime = 90

	// FSGroupID defines the permissions of nfs share volume
	// -----------------------------------------------------
	// NOTE: This feature has been deprecated
	//       Alternative: Use FilePermission 'cas.openebs.io/config' annotation
	//                    key on the backend volume PVC. Sample FilePermissions
	//      	      for FSGID-like configuration --
	//
	//                    name: FilePermissions
	//                    data:
	//                      GID: <group-ID>
	//                      mode: "g+s"
	// -----------------------------------------------------
	FSGroupID = "FSGID"

	// This is the cas-template key for all file permission 'data' keys
	FilePermissions = "FilePermissions"

	// FsUID defines the user owner of the shared directory
	FsUID = "UID"

	// FsGID defines the group owner of the shared directory
	FsGID = "GID"

	// FSMode defines the file permission mode of the shared directory
	FsMode = "mode"

	// NodeAffinityLabels defines the node affinity for the NFS server pod
	NodeAffinityLabels = "NodeAffinityLabels"

	// NFSServerResourceRequests holds key name that represent NFS Resource Requests
	NFSServerResourceRequests = "NFSServerResourceRequests"

	// NFSServerResourceLimits holds key name that represent NFS Resource Limits
	NFSServerResourceLimits = "NFSServerResourceLimits"

	// HookConfigFileName represent file name for hook configuration
	HookConfigFileName = "hook-config"

	// ConfigDirectory defines directory to store config files specific to NFS provisioner
	ConfigDirectory = "/etc/nfs-provisioner"

	// HookConfigFilePath defines path for hook config file
	HookConfigFilePath = ConfigDirectory + "/" + HookConfigFileName
)

const (
	// Some of the PVCs launched with older helm charts, still
	// refer to the StorageClass via beta annotations.
	betaStorageClassAnnotation = "volume.beta.kubernetes.io/storage-class"
)

//GetVolumeConfig creates a new VolumeConfig struct by
// parsing and merging the configuration provided in the PVC
// annotation - cas.openebs.io/config with the
// default configuration of the provisioner.
func (p *Provisioner) GetVolumeConfig(pvName string, pvc *v1.PersistentVolumeClaim) (*VolumeConfig, error) {

	pvConfig := p.defaultConfig

	// extract and merge the cas config from PersistentVolumeClaim
	pvcCASConfigStr := pvc.ObjectMeta.Annotations[string(mconfig.CASConfigKey)]
	klog.V(4).Infof("PVC %v has config:%v", pvc.Name, pvcCASConfigStr)
	if len(strings.TrimSpace(pvcCASConfigStr)) != 0 {
		pvcCASConfig, err := cast.UnMarshallToConfig(pvcCASConfigStr)
		if err == nil {
			pvConfig = cast.MergeConfig(pvcCASConfig, pvConfig)
		} else {
			return nil, errors.Wrapf(err, "failed to get config: invalid config {%v}"+
				" in pvc {%v} in namespace {%v}",
				pvcCASConfigStr, pvc.Name, pvc.Namespace,
			)
		}
	}

	//Fetch the SC
	scName := GetStorageClassNameFromPVC(pvc)
	sc, err := p.kubeClient.StorageV1().StorageClasses().Get(context.TODO(), *scName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get storageclass: missing sc name {%v}", scName)
	}

	// extract and merge the cas config from storageclass
	scCASConfigStr := sc.ObjectMeta.Annotations[string(mconfig.CASConfigKey)]
	klog.V(4).Infof("SC %v has config:%v", *scName, scCASConfigStr)
	if len(strings.TrimSpace(scCASConfigStr)) != 0 {
		scCASConfig, err := cast.UnMarshallToConfig(scCASConfigStr)
		if err == nil {
			// Config keys which already exist (PVC config),
			// will be skipped
			// i.e. PVC config will have precedence over SC config,
			// if both have the same keys
			pvConfig = cast.MergeConfig(pvConfig, scCASConfig)
		} else {
			return nil, errors.Wrapf(err, "failed to get config: invalid sc config {%v}", scCASConfigStr)
		}
	}

	//TODO : extract and merge the cas volume config from pvc
	//This block can be added once validation checks are added
	// as to the type of config that can be passed via PVC
	//pvcCASConfigStr := pvc.ObjectMeta.Annotations[string(mconfig.CASConfigKey)]
	//if len(strings.TrimSpace(pvcCASConfigStr)) != 0 {
	//	pvcCASConfig, err := cast.UnMarshallToConfig(pvcCASConfigStr)
	//	if err == nil {
	//		pvConfig = cast.MergeConfig(pvcCASConfig, pvConfig)
	//	}
	//}

	pvConfigMap, err := cast.ConfigToMap(pvConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read volume config: pvc {%v} in namespace {%v}", pvc.Name, pvc.Namespace)
	}

	listPvConfigMap, err := listConfigToMap(pvConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read volume config: pvc {%v}", pvc.ObjectMeta.Name)
	}

	c := &VolumeConfig{
		pvName:     pvName,
		pvcName:    pvc.ObjectMeta.Name,
		scName:     *scName,
		options:    pvConfigMap,
		configData: dataConfigToMap(pvConfig),
		configList: listPvConfigMap,
	}
	return c, nil
}

// GetNodeAffinityLabels returns NodeAffinity for the NFS server pod
// retreived from NodeAffinity as an array of strings
func (c *VolumeConfig) GetNodeAffinityLabels() (NodeAffinity, error) {
	var nodeAffinity NodeAffinity

	NodeAffinityLabels := c.getList(NodeAffinityLabels)
	if len(NodeAffinityLabels) == 0 {
		return nodeAffinity, nil
	}

	nodeAffinity.MatchExpressions = []v1.NodeSelectorRequirement{
		{
			Key:      "kubernetes.io/hostname",
			Operator: v1.NodeSelectorOpIn,
			Values:   NodeAffinityLabels,
		},
	}

	return nodeAffinity, nil
}

//GetNFSServerTypeFromConfig returns the NFSServerType value configured
// in StorageClass. Default is kernel
func (c *VolumeConfig) GetNFSServerTypeFromConfig() string {
	serverType := c.getValue(KeyPVNFSServerType)
	if len(strings.TrimSpace(serverType)) == 0 {
		return "kernel"
	}
	return serverType
}

//GetBackendStorageClassFromConfig returns the Storage Class
// value configured in StorageClass. Default is ""
func (c *VolumeConfig) GetBackendStorageClassFromConfig() string {
	backingSC := c.getValue(KeyPVBackendStorageClass)
	if len(strings.TrimSpace(backingSC)) == 0 {
		return ""
	}
	return backingSC
}

func (c *VolumeConfig) GetCustomNFSServerConfig() string {
	customServerConfig := c.getValue(CustomServerConfig)
	if len(strings.TrimSpace(customServerConfig)) == 0 {
		return ""
	}
	return customServerConfig
}

func (c *VolumeConfig) GetNFSServerLeaseTime() (int, error) {
	leaseTime := c.getValue(LeaseTime)
	if len(strings.TrimSpace(leaseTime)) == 0 {
		return DefaultLeaseTime, nil
	}
	leaseTimeVal, err := strconv.Atoi(leaseTime)
	if err != nil {
		return 0, err
	}
	if leaseTimeVal == 0 {
		leaseTimeVal = DefaultLeaseTime
	}

	return leaseTimeVal, nil
}

func (c *VolumeConfig) GetNFServerGraceTime() (int, error) {
	graceTime := c.getValue(GraceTime)
	if len(strings.TrimSpace(graceTime)) == 0 {
		return DefaultGraceTime, nil
	}
	graceTimeVal, err := strconv.Atoi(graceTime)
	if err != nil {
		return 0, err
	}

	if graceTimeVal == 0 {
		graceTimeVal = DefaultGraceTime
	}
	return graceTimeVal, nil
}

// GetFSGroupID fetches the group ID permissions from
// StorageClass if specified
// -----------------------------------------------------
// NOTE: This feature has been deprecated
//       Alternative: Use FilePermission 'cas.openebs.io/config' annotation
//                    key on the backend volume PVC. Sample FilePermissions
//      	      for FSGID-like configuration --
//
//                    name: FilePermissions
//                    data:
//                      GID: <group-ID>
//                      mode: "g+s"
// -----------------------------------------------------
func (c *VolumeConfig) GetFSGroupID() (*int64, error) {
	fsGroupIDStr := c.getValue(FSGroupID)
	if len(strings.TrimSpace(fsGroupIDStr)) == 0 {
		return nil, nil
	}
	fsGIDInt, err := strconv.ParseInt(fsGroupIDStr, 10, 64)
	if err != nil {
		return nil, err
	}

	klog.Infof("The %s option key '%s' is being deprecated"+
		" and will be removed in future releases."+
		"\nYou may use the %s option key in the "+
		"NFS PersistentVolumeClaim's or NFS StorageClass's "+
		"(NFS PVC's configuration takes precedence) %s "+
		"annotation key to achieve the same result."+
		"\nSample config:\n"+
		"\t\t"+"- name: FilePermissions\n"+
		"\t\t"+"  data:\n"+
		"\t\t"+"    %s: \"%s\"\n"+
		"\t\t"+"    %s: \"g+s\"\n",
		string(mconfig.CASConfigKey), FSGroupID, FilePermissions,
		string(mconfig.CASConfigKey), FsGID, fsGroupIDStr, FsMode,
	)
	return &fsGIDInt, nil
}

// GetFsGID fetches the group owner's ID from
// PVC annotation, if specified
func (c *VolumeConfig) GetFsGID() (string, error) {
	fsGIDStr := strings.TrimSpace(c.getData(FilePermissions, FsGID))

	// TODO: remove this block when FSGID is deprecated
	deprecatedFsGroupIDStr, _ := c.GetFSGroupID()

	existsFsGIDStr := (len(fsGIDStr) > 0)
	existsDeprecatedFsGroupIDStr := (deprecatedFsGroupIDStr != nil)

	// Checking if FSGID and FilePermissions (GID) are being used together
	if existsFsGIDStr && existsDeprecatedFsGroupIDStr {
		return "", errors.Errorf("both '%s' and '%s."+
			"%s' cannot be used together",
			FSGroupID, FilePermissions, FsGID,
		)
	}

	if existsDeprecatedFsGroupIDStr {
		return "", nil
	}

	// existsFsGIDStr == true OR fsGIDStr == ""
	return fsGIDStr, nil
}

// GetFsGID fetches the user owner's ID from
// PVC annotation, if specified
func (c *VolumeConfig) GetFsUID() string {
	fsUIDStr := strings.TrimSpace(c.getData(FilePermissions, FsUID))
	if len(fsUIDStr) == 0 {
		return ""
	}

	return fsUIDStr
}

// GetFsMode fetches the file mode from PVC
// or StorageClass annotation, if specified
func (c *VolumeConfig) GetFsMode() (string, error) {
	fsModeStr := strings.TrimSpace(c.getData(FilePermissions, FsMode))

	// TODO: remove this block when FSGID is deprecated
	deprecatedFsGroupIDStr, _ := c.GetFSGroupID()

	existsFsModeStr := (len(fsModeStr) > 0)
	existsDeprecatedFsGroupIDStr := (deprecatedFsGroupIDStr != nil)

	// Checking if FSGID and FilePermissions (mode) are being used together
	if existsFsModeStr && existsDeprecatedFsGroupIDStr {
		return "", errors.Errorf("both '%s' and '%s."+
			"%s' cannot be used together",
			FSGroupID, FilePermissions, FsMode,
		)
	}

	if existsDeprecatedFsGroupIDStr {
		return "", nil
	}

	// existsFsModeStr == true OR fsModeStr == ""
	return fsModeStr, nil
}

// GetNFSServerResourceRequirements fetches the resource(cpu & memory) request &
// limits for NFS server from StorageClass only if specified
func (c *VolumeConfig) GetNFSServerResourceRequirements() (*v1.ResourceRequirements, error) {
	var err error
	resourceRequirements := &v1.ResourceRequirements{}
	resourceRequirements.Requests, err = c.getResourceList(NFSServerResourceRequests)
	if err != nil {
		return nil, err
	}

	resourceRequirements.Limits, err = c.getResourceList(NFSServerResourceLimits)
	if err != nil {
		return nil, err
	}
	return resourceRequirements, nil
}

// getResourceList is a utility function to extract resource list
// and convert from map[string]interface{} to proper Go struct
func (c *VolumeConfig) getResourceList(key string) (v1.ResourceList, error) {
	var resourceList v1.ResourceList
	dataStr := c.getValue(key)
	err := yaml.Unmarshal([]byte(dataStr), &resourceList)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal data %s", dataStr)
	}
	return resourceList, nil
}

//getValue is a utility function to extract the value
// of the `key` from the ConfigMap object - which is
// map[string]interface{map[string][string]}
// Example:
// {
//     key1: {
//             value: value1
//             enabled: true
//           }
// }
// In the above example, if `key1` is passed as input,
//   `value1` will be returned.
func (c *VolumeConfig) getValue(key string) string {
	if configObj, ok := util.GetNestedField(c.options, key).(map[string]string); ok {
		if val, p := configObj[string(mconfig.ValuePTP)]; p {
			return val
		}
	}
	return ""
}

// This gets the list of values for the 'List' parameter.
func (c *VolumeConfig) getList(key string) []string {
	if listValues, ok := util.GetNestedField(c.configList, key).([]string); ok {
		return listValues
	}
	//Default case
	return nil
}

//getData is a utility function to extract the value
// of the `key` from the ConfigMap object - which is
// map[string]interface{map[string]interface{map[string]string}}
// Example:
// {
//     key1: {
//             value: value1
//             data: {
//                     dataKey1: dataValue1
//                   }
//           }
// }
// In the above example, if `key1` and `dataKey1` are passed as input,
//   `dataValue1` will be returned.
func (c *VolumeConfig) getData(key string, dataKey string) string {
	if configData, ok := util.GetNestedField(c.configData, key).(map[string]string); ok {
		if val, p := configData[dataKey]; p {
			return val
		}
	}
	//Default case
	return ""
}

// GetStorageClassNameFromPVC extracts the StorageClass name from PVC
func GetStorageClassNameFromPVC(pvc *v1.PersistentVolumeClaim) *string {
	// Use beta annotation first
	class, found := pvc.Annotations[betaStorageClassAnnotation]
	if found {
		return &class
	}
	return pvc.Spec.StorageClassName
}

// GetNFSServerTypeFromPV extracts the NFS Server Type name from PV
func GetNFSServerTypeFromPV(pv *v1.PersistentVolume) string {
	//TODO extract this from PV annotations
	return "kernel"
}

// hookConfigFileExist check if hook config file exists or not
func hookConfigFileExist() (bool, error) {
	// HookConfigFilePath
	_, err := os.Stat(HookConfigFilePath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// initializeHook read the hook config file and update the given hook variable
// return value:
// 	- nil
// 		- If hook config file doesn't exists
// 		- If hook config file is parsed and given hook variable is updated
// 	- error
// 		- If hook config is invalid
func initializeHook(hook **nfshook.Hook) error {
	hookFileExists, err := hookConfigFileExist()
	if err != nil {
		return errors.Errorf("failed to check hook config file, err=%s", err)
	}

	if !hookFileExists {
		return nil
	}

	data, err := ioutil.ReadFile(HookConfigFilePath)
	if err != nil {
		return errors.Errorf("failed to read hook config file, err=%s", err)
	}

	hookObj, err := nfshook.ParseHooks(data)
	if err != nil {
		return err
	}

	*hook = hookObj
	return nil
}

func dataConfigToMap(pvConfig []mconfig.Config) map[string]interface{} {
	m := map[string]interface{}{}

	for _, configObj := range pvConfig {
		//No Data Parameter
		if configObj.Data == nil {
			continue
		}

		configName := strings.TrimSpace(configObj.Name)
		m[configName] = configObj.Data
	}

	return m
}

func listConfigToMap(pvConfig []mconfig.Config) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	for _, configObj := range pvConfig {
		//No List Parameter
		if len(configObj.List) == 0 {
			continue
		}

		configName := strings.TrimSpace(configObj.Name)
		confHierarchy := map[string]interface{}{
			configName: configObj.List,
		}
		isMerged := util.MergeMapOfObjects(m, confHierarchy)
		if !isMerged {
			return nil, errors.Errorf("failed to transform cas config 'List' for configName '%s' to map: failed to merge: %s", configName, configObj)
		}
	}

	return m, nil
}
