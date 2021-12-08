/*
Copyright 2021 The OpenEBS Authors.

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

package tests

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// Checks if all of the input ENV's present in the
// container exist and have non-empty values
// Returns true iff all of the ENV's values are non-empty
func isEnvValuePresent(k8sContainer *corev1.Container, envList ...string) (bool, error) {
	envListLen := len(envList)
	if k8sContainer == nil || envListLen == 0 {
		return false, errors.Errorf("failed to check for ENVs: invalid input")
	}

	containerEnvList := make(map[string]int)

	for i, env := range k8sContainer.Env {
		containerEnvList[env.Name] = i
	}

	for _, env := range envList {
		if _, ok := containerEnvList[env]; !ok {
			return false, nil
		}
	}

	return true, nil
}

// Checks if all of the input ENV's present in the
// container exist and have their corresponding values
// Returns true iff all of the ENVs have their corresponding values
func isEnvValueCorrect(k8sContainer *corev1.Container, envVal map[string]string) (bool, error) {
	if k8sContainer == nil || envVal == nil {
		return false, errors.Errorf("failed to check for ENVs: invalid input")
	}

	containerEnvList := make(map[string]string)

	for _, env := range k8sContainer.Env {
		containerEnvList[env.Name] = env.Value
	}

	for env, val := range envVal {
		if containerVal, ok := containerEnvList[env]; !ok || containerVal != val {
			return false, nil
		}
	}

	return true, nil
}
