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

package hook

import (
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// ParseHooks will parse the given data and return generated Hook object
func ParseHooks(hookData []byte) (*Hook, error) {
	var hook Hook
	err := yaml.Unmarshal(hookData, &hook)
	if err != nil {
		return nil, errors.Wrapf(err, "error Unmarshalling hookData")
	}

	h := &hook
	h.updateAvailableActions()
	return h, nil
}

func (h *Hook) updateAvailableActions() {
	h.availableActions = make(map[ProvisionerEventType]map[int]struct{})
	h.availableActions[ProvisionerEventCreate] = make(map[int]struct{})
	h.availableActions[ProvisionerEventDelete] = make(map[int]struct{})

	for _, cfg := range h.Config {
		if cfg.BackendPVCConfig != nil {
			h.availableActions[cfg.Event][ResourceBackendPVC] = struct{}{}
		}

		if cfg.BackendPVConfig != nil {
			h.availableActions[cfg.Event][ResourceBackendPV] = struct{}{}
		}

		if cfg.NFSServiceConfig != nil {
			h.availableActions[cfg.Event][ResourceNFSService] = struct{}{}
		}

		if cfg.NFSPVConfig != nil {
			h.availableActions[cfg.Event][ResourceNFSPV] = struct{}{}
		}

		if cfg.NFSDeploymentConfig != nil {
			h.availableActions[cfg.Event][ResourceNFSServerDeployment] = struct{}{}
		}
	}
	return
}
