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

	"github.com/openebs/dynamic-nfs-provisioner/pkg/helper"
)

// deploymentHookAction will execute the given hook config on the object for the given action
func deploymentHookAction(hookCfg *DeploymentHook, action ActionOp, obj interface{}) error {
	if hookCfg == nil {
		return nil
	}

	dObj, ok := obj.(*appsv1.Deployment)
	if !ok {
		return errors.Errorf("%T is not a Deployment type", obj)
	}

	switch action {
	case ActionOpAddOrUpdate:
		deploymentHookActionAdd(dObj, *hookCfg)
	case ActionOpRemove:
		deploymentHookActionRemove(dObj, *hookCfg)
	}

	return nil
}

// deploymentHookActionAdd will add the given hook config to the given object
func deploymentHookActionAdd(obj *appsv1.Deployment, hookCfg DeploymentHook) {
	if len(hookCfg.Annotations) != 0 {
		AddAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		helper.AddFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}

// deploymentHookActionRemove will remove the given hook config to the given object
func deploymentHookActionRemove(obj *appsv1.Deployment, hookCfg DeploymentHook) {
	if len(hookCfg.Annotations) != 0 {
		helper.RemoveAnnotations(&obj.ObjectMeta, hookCfg.Annotations)
	}

	if len(hookCfg.Finalizers) != 0 {
		helper.RemoveFinalizers(&obj.ObjectMeta, hookCfg.Finalizers)
	}
	return
}
