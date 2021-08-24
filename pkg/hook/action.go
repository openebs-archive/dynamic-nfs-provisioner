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

// Action run hooks for the given object type as per the event
// Action will skip further hook execution if any error occurred
func (h *Hook) Action(obj interface{}, resourceType int, eventType ProvisionerEventType) error {
	var err error
	for _, cfg := range h.Config {
		if cfg.Event != eventType {
			continue
		}

		switch resourceType {
		case ResourceBackendPVC:
			err = pvc_hook_action(cfg.BackendPVCConfig, cfg.Action, obj)
			if err != nil {
				return err
			}
		case ResourceBackendPV:
			err = pv_hook_action(cfg.BackendPVConfig, cfg.Action, obj)
			if err != nil {
				return err
			}
		case ResourceNFSService:
			err = service_hook_action(cfg.NFSServiceConfig, cfg.Action, obj)
			if err != nil {
				return err
			}
		case ResourceNFSPV:
			err = pv_hook_action(cfg.NFSPVConfig, cfg.Action, obj)
			if err != nil {
				return err
			}
		case ResourceNFSServerDeployment:
			err = deployment_hook_action(cfg.NFSDeploymentConfig, cfg.Action, obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ActionExists will check if action exists for the give resource type and event type
func (h *Hook) ActionExists(resourceType int, eventType ProvisionerEventType) bool {
	_, actionExist := h.availableActions[eventType][resourceType]
	return actionExist
}
