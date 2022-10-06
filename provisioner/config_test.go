/*
Copyright 2021 The OpenEBS Authors

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
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetResourceList(t *testing.T) {
	tests := map[string]struct {
		volumeConfig         *VolumeConfig
		key                  string
		expectedResourceList corev1.ResourceList
		isErrExpected        bool
	}{
		"When NFS resource requests has only memory field": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					NFSServerResourceRequests: map[string]string{
						"value": func() string {
							resourceList := make(map[corev1.ResourceName]resource.Quantity)
							resourceList[corev1.ResourceMemory] = resource.MustParse("500M")
							data, err := yaml.Marshal(resourceList)
							if err != nil {
								t.Errorf("failed to convert to YAML error %v", err)
							}
							return string(data)
						}(),
					},
				},
			},
			key: NFSServerResourceRequests,
			expectedResourceList: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("500M"),
			},
		},
		"When NFS resource requests has both memory and cpu": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					NFSServerResourceRequests: map[string]string{
						"value": func() string {
							resourceList := make(map[corev1.ResourceName]resource.Quantity)
							resourceList[corev1.ResourceMemory] = resource.MustParse("500M")
							resourceList[corev1.ResourceCPU] = resource.MustParse("500m")
							data, err := yaml.Marshal(resourceList)
							if err != nil {
								t.Errorf("failed to convert to YAML error %v", err)
							}
							return string(data)
						}(),
					},
				},
			},
			key: NFSServerResourceRequests,
			expectedResourceList: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("500M"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
		"When NFS resource limits has only memory specified": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					"NFSResourceLimits": map[string]string{
						"value": func() string {
							resourceList := make(map[corev1.ResourceName]resource.Quantity)
							resourceList[corev1.ResourceMemory] = resource.MustParse("150Mi")
							data, err := yaml.Marshal(resourceList)
							if err != nil {
								t.Errorf("failed to convert to YAML error %v", err)
							}
							return string(data)
						}(),
					},
				},
			},
			key: "NFSResourceLimits",
			expectedResourceList: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("150Mi"),
			},
		},
		"When NFS resource limits has both memory & cpu specified": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					"NFSResourceLimits": map[string]string{
						"value": func() string {
							resourceList := make(map[corev1.ResourceName]resource.Quantity)
							resourceList[corev1.ResourceMemory] = resource.MustParse("150Mi")
							resourceList[corev1.ResourceCPU] = resource.MustParse("150m")
							data, err := yaml.Marshal(resourceList)
							if err != nil {
								t.Errorf("failed to convert to YAML error %v", err)
							}
							return string(data)
						}(),
					},
				},
			},
			key: "NFSResourceLimits",
			expectedResourceList: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: resource.MustParse("150Mi"),
				corev1.ResourceCPU:    resource.MustParse("150m"),
			},
		},
		"When NFS resource limits are specified but requesting for requests": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					"NFSResourceLimits": map[string]string{
						"value": func() string {
							resourceList := make(map[corev1.ResourceName]resource.Quantity)
							resourceList[corev1.ResourceMemory] = resource.MustParse("150Mi")
							resourceList[corev1.ResourceCPU] = resource.MustParse("150m")
							data, err := yaml.Marshal(resourceList)
							if err != nil {
								t.Errorf("failed to convert to YAML error %v", err)
							}
							return string(data)
						}(),
					},
				},
			},
			key:                  NFSServerResourceRequests,
			expectedResourceList: nil,
		},
		"When invalid NFS resource requests are specified error should occur": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					NFSServerResourceRequests: map[string]string{
						"value": `memory: 50Ci`,
					},
				},
			},
			key:           NFSServerResourceRequests,
			isErrExpected: true,
		},
	}

	for name, test := range tests {
		gotOutput, err := test.volumeConfig.getResourceList(test.key)
		if test.isErrExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got %v", name, err)
		}
		if !test.isErrExpected {
			if !reflect.DeepEqual(test.expectedResourceList, gotOutput) {
				t.Errorf("%q test has following diff %s", name, cmp.Diff(test.expectedResourceList, gotOutput))
			}
		}
	}
}

func Test_listConfigToMap(t *testing.T) {
	tests := map[string]struct {
		pvConfig      []mconfig.Config
		expectedValue map[string]interface{}
		isErrExpected bool
	}{
		"Valid list parameter": {
			pvConfig: []mconfig.Config{
				{Name: "NodeAffinityLabels", List: []string{"node1", "node2"}},
			},
			expectedValue: map[string]interface{}{
				"NodeAffinityLabels": []string{"node1", "node2"},
			},
			isErrExpected: false,
		},
	}
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			got, err := listConfigToMap(v.pvConfig)
			if (err != nil) != v.isErrExpected {
				t.Errorf("listConfigToMap() error = %v, wantErr %v", err, v.isErrExpected)
				return
			}
			if !reflect.DeepEqual(got, v.expectedValue) {
				t.Errorf("listConfigToMap() got = %v, want %v", got, v.expectedValue)
			}
		})
	}
}

func TestGetNodeAffinityList(t *testing.T) {
	tests := map[string]struct {
		pvConfig       []mconfig.Config
		volumeConfig   *VolumeConfig
		key            string
		expectedOutput NodeAffinity
		isErrExpected  bool
	}{
		"When a valid node affinity is used": {
			volumeConfig: &VolumeConfig{
				configList: map[string]interface{}{
					NodeAffinityLabels: []string{"node1", "node2"},
				},
			},
			expectedOutput: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/hostname",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"node1", "node2"},
					},
				},
			},
			key:           NodeAffinityLabels,
			isErrExpected: false,
		},
		"When an empty node affinity is used": {
			volumeConfig: &VolumeConfig{
				configList: map[string]interface{}{
					NodeAffinityLabels: nil,
				},
			},
			expectedOutput: NodeAffinity{},
			isErrExpected:  false,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		gotOutput, err := test.volumeConfig.GetNodeAffinityLabels()

		if test.isErrExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got %v", name, err)
		}
		if !test.isErrExpected {
			if !reflect.DeepEqual(test.expectedOutput, gotOutput) {
				t.Errorf("%q test has following diff %s", name, cmp.Diff(test.expectedOutput, gotOutput))
			}
		}
	}
}

func TestGetFsGID(t *testing.T) {
	tests := map[string]struct {
		volumeConfig   *VolumeConfig
		expectedOutput string
		isErrExpected  bool
	}{
		"When to-be-deprecated FSGID and FilePermissions-GID are used together": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					FSGroupID: map[string]string{
						string(mconfig.ValuePTP): "1000",
					},
				},
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsGID: "2000",
					},
				},
			},
			expectedOutput: "",
			isErrExpected:  true,
		},
		"When a valid FilePermissions-GID is used": {
			volumeConfig: &VolumeConfig{
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsGID: "2000",
					},
				},
			},
			expectedOutput: "2000",
			isErrExpected:  false,
		},
		"When an empty FilePermissions-GID is used": {
			volumeConfig: &VolumeConfig{
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsGID: "",
					},
				},
			},
			expectedOutput: "",
			isErrExpected:  false,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		gotOutput, err := test.volumeConfig.GetFsGID()
		if test.isErrExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got %v", name, err)
		}
		if !test.isErrExpected {
			if !(test.expectedOutput == gotOutput) {
				t.Errorf("%q test: expected %s, but got %s", name, test.expectedOutput, gotOutput)
			}
		}
	}
}

func TestGetFsMode(t *testing.T) {
	tests := map[string]struct {
		volumeConfig   *VolumeConfig
		expectedOutput string
		isErrExpected  bool
	}{
		"When to-be-deprecated FSGID and FilePermissions-mode are used together": {
			volumeConfig: &VolumeConfig{
				options: map[string]interface{}{
					FSGroupID: map[string]string{
						string(mconfig.ValuePTP): "1000",
					},
				},
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsMode: "0744",
					},
				},
			},
			expectedOutput: "",
			isErrExpected:  true,
		},
		"When a valid FilePermissions-mode is used": {
			volumeConfig: &VolumeConfig{
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsMode: "0744",
					},
				},
			},
			expectedOutput: "0744",
			isErrExpected:  false,
		},
		"When an empty FilePermissions-mode is used": {
			volumeConfig: &VolumeConfig{
				configData: map[string]interface{}{
					FilePermissions: map[string]string{
						FsMode: "",
					},
				},
			},
			expectedOutput: "",
			isErrExpected:  false,
		},
	}

	for name, test := range tests {
		name := name
		test := test
		gotOutput, err := test.volumeConfig.GetFsMode()
		if test.isErrExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got %v", name, err)
		}
		if !test.isErrExpected {
			if !(test.expectedOutput == gotOutput) {
				t.Errorf("%q test: expected %s, but got %s", name, test.expectedOutput, gotOutput)
			}
		}
	}
}
