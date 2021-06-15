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
	"strings"

	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	cast "github.com/openebs/maya/pkg/castemplate/v1alpha1"
	"github.com/openebs/maya/pkg/util"
	"k8s.io/klog"

	//"github.com/pkg/errors"
	errors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//storagev1 "k8s.io/api/storage/v1"
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

	// LeaseTime defines the renewl period(in seconds) for client state
	// if not set then default value(90s) will be used
	LeaseTime        = "LeaseTime"
	DefaultLeaseTime = 90

	// GraceTime defines the recovery period(in seconds) to reclaim locks
	// If it is not set then default value(90s) will be used
	GraceTime        = "GraceTime"
	DefaultGraceTime = 90
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

	//Fetch the SC
	scName := GetStorageClassNameFromPVC(pvc)
	sc, err := p.kubeClient.StorageV1().StorageClasses().Get(*scName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get storageclass: missing sc name {%v}", scName)
	}

	// extract and merge the cas config from storageclass
	scCASConfigStr := sc.ObjectMeta.Annotations[string(mconfig.CASConfigKey)]
	klog.V(4).Infof("SC %v has config:%v", *scName, scCASConfigStr)
	if len(strings.TrimSpace(scCASConfigStr)) != 0 {
		scCASConfig, err := cast.UnMarshallToConfig(scCASConfigStr)
		if err == nil {
			pvConfig = cast.MergeConfig(scCASConfig, pvConfig)
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
		return nil, errors.Wrapf(err, "unable to read volume config: pvc {%v}", pvc.ObjectMeta.Name)
	}

	c := &VolumeConfig{
		pvName:  pvName,
		pvcName: pvc.ObjectMeta.Name,
		scName:  *scName,
		options: pvConfigMap,
	}
	return c, nil
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
