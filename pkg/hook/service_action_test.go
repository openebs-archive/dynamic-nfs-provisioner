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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestService_hook_action(t *testing.T) {
	tests := []struct {
		name        string
		hook        *ServiceHook
		obj         interface{}
		expectedObj interface{}
		actionType  ActionOp
	}{
		{
			name:        "when service hook is nil, object should not be modified",
			hook:        nil,
			obj:         generateFakeServiceObj("ns1", "name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakeServiceObj("ns1", "name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  ActionOpAddOrUpdate,
		},
		{
			name:        "when service hook is configured to add metadata, object should be modified",
			hook:        buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			obj:         generateFakeServiceObj("ns2", "name2", nil, nil),
			expectedObj: generateFakeServiceObj("ns2", "name2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  ActionOpAddOrUpdate,
		},
		{
			name:        "when service hook is configured to remove metadata, object should be modified",
			hook:        buildServiceHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			obj:         generateFakeServiceObj("ns3", "name3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakeServiceObj("ns3", "name3", map[string]string{}, []string{}),
			actionType:  ActionOpRemove,
		},
		{
			name:        "when service hook is configured to remove non-existing metadata, object should not be modified",
			hook:        buildServiceHook(map[string]string{"test.com/key": "val"}, []string{"test.com/finalizer"}),
			obj:         generateFakeServiceObj("ns4", "name4", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakeServiceObj("ns4", "name4", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  ActionOpRemove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "object should not be nil")
			err := service_hook_action(test.hook, test.actionType, test.obj)
			assert.Nil(t, err, "service_hook_action returned error")
			assert.Equal(t, test.expectedObj, test.obj, "object should match")
		})
	}
}

// TestService_hook_action_var verifies if template variables are processed or not
func TestService_hook_action_var(t *testing.T) {
	tests := []struct {
		name        string
		hook        *ServiceHook
		obj         interface{}
		expectedObj interface{}
		actionType  ActionOp
	}{
		{
			name:       "when Service hook is configured to add template annotation, object should be modified",
			hook:       buildServiceHook(map[string]string{"test.com/key": string(TemplateVarCurrentTime)}, nil),
			obj:        generateFakeServiceObj("ns1", "name1", nil, nil),
			actionType: ActionOpAddOrUpdate,
		},
		{
			name:        "when Service hook is configured to remove template annotation, annotation should be removed",
			hook:        buildServiceHook(map[string]string{"test.io/key": string(TemplateVarCurrentTime)}, []string{"test.io/finalizer"}),
			obj:         generateFakeServiceObj("ns2", "name2", map[string]string{"test.io/key": "2021-09-23T15:27:04+05:30"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakeServiceObj("ns2", "name2", map[string]string{}, []string{}),
			actionType:  ActionOpRemove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "object should not be nil")
			err := service_hook_action(test.hook, test.actionType, test.obj)
			assert.Nil(t, err, "service_hook_action returned error")
			if test.expectedObj != nil {
				assert.Equal(t, test.expectedObj, test.obj, "object should match")
			} else {
				obj, _ := test.obj.(metav1.ObjectMeta)
				for _, v := range obj.Annotations {
					_, err := time.Parse(v, time.RFC3339)
					assert.Nil(t, err, "Annotation value is having invalid timestamp")
				}
			}
		})
	}
}
