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

	"github.com/openebs/dynamic-nfs-provisioner/pkg/helper"
)

// serviceHookAction will execute the given hook config on the object for the given action
func serviceHookAction(hookCfg *ServiceHook, action ActionOp, obj interface{}) error {
	if hookCfg == nil {
		return nil
	}

	sObj, ok := obj.(*corev1.Service)
	if !ok {
		return errors.Errorf("%T is not a Service type", obj)
	}

	switch action {
	case ActionOpAddOrUpdate:
		serviceHookActionAdd(sObj, *hookCfg)
	case ActionOpRemove:
		serviceHookActionRemove(sObj, *hookCfg)
	}

	return nil
}

// serviceHookActionAdd will add the given hook config to the given object
func serviceHookActionAdd(obj *corev1.Service, hookCfg ServiceHook) {
	if len(hookCfg.Annotations) != 0 {
		AddAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		helper.AddFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}

// serviceHookActionRemove will remove the given hook config to the given object
func serviceHookActionRemove(obj *corev1.Service, hookCfg ServiceHook) {
	if len(hookCfg.Annotations) != 0 {
		helper.RemoveAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		helper.RemoveFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}
