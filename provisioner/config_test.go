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
