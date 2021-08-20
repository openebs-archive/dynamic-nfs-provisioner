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
	corev1 "k8s.io/api/core/v1"
)

// service_hook_action will execute the given hook config on the object for the given action
func service_hook_action(hookCfg *ServiceHook, action HookActionType, obj interface{}) error {
	if hookCfg == nil {
		return nil
	}

	sObj, ok := obj.(*corev1.Service)
	if !ok {
		return errors.Errorf("%T is not a Service type", obj)
	}

	switch action {
	case HookActionAdd:
		service_hook_action_add(sObj, *hookCfg)
	case HookActionRemove:
		service_hook_action_remove(sObj, *hookCfg)
	}

	return nil
}

// service_hook_action_add will add the given hook config to the given object
func service_hook_action_add(obj *corev1.Service, hookCfg ServiceHook) {
	if len(hookCfg.Annotations) != 0 {
		addAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		addFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}

// service_hook_action_remove will remove the given hook config to the given object
func service_hook_action_remove(obj *corev1.Service, hookCfg ServiceHook) {
	if len(hookCfg.Annotations) != 0 {
		removeAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		removeFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}
