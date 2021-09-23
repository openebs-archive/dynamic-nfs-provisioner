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

	if hook.Version != HookVersion {
		return nil, errors.Errorf("Hook Version=%s not supported", hook.Version)
	}

	h := &hook
	h.updateAvailableActions()
	return h, nil
}

func (h *Hook) updateAvailableActions() {
	h.availableActions = make(map[EventType]map[int]struct{})
	h.availableActions[EventTypeCreateVolume] = make(map[int]struct{})
	h.availableActions[EventTypeDeleteVolume] = make(map[int]struct{})

	for actionType, cfg := range h.Config {
		actionEvent, ok := ActionForEventMap[actionType]
		if !ok {
			continue
		}

		if cfg.BackendPVCConfig != nil {
			h.availableActions[actionEvent.evType][ResourceBackendPVC] = struct{}{}
		}

		if cfg.BackendPVConfig != nil {
			h.availableActions[actionEvent.evType][ResourceBackendPV] = struct{}{}
		}

		if cfg.NFSServiceConfig != nil {
			h.availableActions[actionEvent.evType][ResourceNFSService] = struct{}{}
		}

		if cfg.NFSPVConfig != nil {
			h.availableActions[actionEvent.evType][ResourceNFSPV] = struct{}{}
		}

		if cfg.NFSDeploymentConfig != nil {
			h.availableActions[actionEvent.evType][ResourceNFSServerDeployment] = struct{}{}
		}
	}
	return
}
