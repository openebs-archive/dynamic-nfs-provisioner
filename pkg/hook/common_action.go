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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// addFinalizers add the given finalizers to the given meta object
// Finalizer will be added only if it doesn't exist in object
func addFinalizers(objMeta *metav1.ObjectMeta, finalizers []string) {
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

// addAnnotations add given annotations to given meta object
// Annotation will be added only if it doesn't exist in object
func addAnnotations(objMeta *metav1.ObjectMeta, annotations map[string]string) {
	if objMeta.Annotations == nil {
		objMeta.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		objMeta.Annotations[k] = v
	}
	return
}

// removeFinalizers remove the given finalizers from the given meta object
func removeFinalizers(objMeta *metav1.ObjectMeta, finalizers []string) {
	for _, f := range finalizers {
		for i := 0; i < len(objMeta.Finalizers); i++ {
			if objMeta.Finalizers[i] == f {
				objMeta.Finalizers = append(objMeta.Finalizers[:i], objMeta.Finalizers[i+1:]...)
				break
			}
		}
	}
}

// removeAnnotations remove the given annotations from given meta object
func removeAnnotations(objMeta *metav1.ObjectMeta, annotations map[string]string) {
	for k := range annotations {
		delete(objMeta.Annotations, k)
	}
	return
}
