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

package helper

import (
	"encoding/json"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// AddFinalizers add the given finalizers to the given meta object
// Finalizer will be added only if it doesn't exist in object
func AddFinalizers(objMeta *metav1.ObjectMeta, finalizers []string) {
	for _, f := range finalizers {
		var finalizerExists bool
		for _, existingF := range objMeta.Finalizers {
			if f == existingF {
				finalizerExists = true
				break
			}
		}

		if !finalizerExists {
			objMeta.Finalizers = append(objMeta.Finalizers, f)
		}
	}
}

// AddAnnotations add given annotations to given meta object
// Object annotations will be overridden if any of the given annotations exists with the same key
func AddAnnotations(objMeta *metav1.ObjectMeta, annotations map[string]string) {
	if objMeta.Annotations == nil {
		objMeta.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		objMeta.Annotations[k] = v
	}
	return
}

// RemoveFinalizers remove the given finalizers from the given meta object
func RemoveFinalizers(objMeta *metav1.ObjectMeta, finalizers []string) {
	for _, f := range finalizers {
		for i := 0; i < len(objMeta.Finalizers); i++ {
			if objMeta.Finalizers[i] == f {
				objMeta.Finalizers = append(objMeta.Finalizers[:i], objMeta.Finalizers[i+1:]...)
				break
			}
		}
	}
}

// RemoveAnnotations remove the given annotations from given meta object
func RemoveAnnotations(objMeta *metav1.ObjectMeta, annotations map[string]string) {
	for k := range annotations {
		delete(objMeta.Annotations, k)
	}
	return
}

// GetPatchData will return the diff data for the given objects
func GetPatchData(oldObj, newObj interface{}) ([]byte, []byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, nil, errors.Errorf("marshal old object failed: %v", err)
	}

	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, nil, errors.Errorf("marshal new object failed: %v", err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, oldObj)
	if err != nil {
		return nil, nil, errors.Errorf("CreateTwoWayMergePatch failed: %v", err)
	}

	return patchBytes, oldData, nil
}
