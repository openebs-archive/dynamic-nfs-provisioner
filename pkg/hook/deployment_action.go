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
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
)

// deployment_hook_action will execute the given hook config on the object for the given action
func deployment_hook_action(hookCfg *DeploymentHook, action HookActionType, obj interface{}) error {
	if hookCfg == nil {
		return nil
	}

	dObj, ok := obj.(*appsv1.Deployment)
	if !ok {
		return errors.Errorf("%T is not a Deployment type", obj)
	}

	switch action {
	case HookActionAdd:
		deployment_hook_action_add(dObj, *hookCfg)
	case HookActionRemove:
		deployment_hook_action_remove(dObj, *hookCfg)
	}

	return nil
}

// deployment_hook_action_add will add the given hook config to the given object
func deployment_hook_action_add(obj *appsv1.Deployment, hookCfg DeploymentHook) {
	if len(hookCfg.Annotations) != 0 {
		addAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		addFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}

// deployment_hook_action_remove will remove the given hook config to the given object
func deployment_hook_action_remove(obj *appsv1.Deployment, hookCfg DeploymentHook) {
	if len(hookCfg.Annotations) != 0 {
		removeAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		removeFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}
