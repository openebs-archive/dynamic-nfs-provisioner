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
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddAnnotations add given annotations to given meta object
// Object annotations will be overridden if any of the given annotations exists with the same key
func AddAnnotations(objMeta *metav1.ObjectMeta, annotations map[string]string) {
	if objMeta.Annotations == nil {
		objMeta.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		idx := strings.Index(v, string(TemplateVarCurrentTime))
		if idx == -1 {
			objMeta.Annotations[k] = v
		} else {
			objMeta.Annotations[k] = strings.ReplaceAll(v, string(TemplateVarCurrentTime), time.Now().Format(time.RFC3339))
		}
	}
	return
}
