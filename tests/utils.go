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
// container exist and have their corresponding values
// Returns true iff all of the ENVs have their corresponding values
func isEnvValueCorrect(k8sContainer *corev1.Container, envVal map[string]string) (bool, error) {
	if k8sContainer == nil || envVal == nil {
		return false, errors.Errorf("failed to check for ENVs: invalid input")
	}

	envMatchCount := 0
	for _, containerENV := range k8sContainer.Env {
		if val, ok := envVal[containerENV.Name]; ok && containerENV.Value == val {
			envMatchCount++
		}
	}

	return envMatchCount == len(envVal), nil
}
