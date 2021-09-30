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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateFakeObjMeta(finalizer []string, annotations map[string]string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Finalizers:  finalizer,
		Annotations: annotations,
	}
}

func generateFakeServiceObj(namespace, name string, annotations map[string]string, finalizers []string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Finalizers:  finalizers,
			Annotations: annotations,
		},
	}
}

func TestAddFinalizers(t *testing.T) {
	tests := []struct {
		name        string
		obj         *metav1.ObjectMeta
		finalizers  []string
		expectedObj *metav1.ObjectMeta
	}{
		{
			name:        "given finalizer should be added",
			finalizers:  []string{"test.io/test"},
			obj:         generateFakeObjMeta(nil, nil),
			expectedObj: generateFakeObjMeta([]string{"test.io/test"}, nil),
		},
		{
			name:        "if finalizer list is empty, obj should not be modified",
			finalizers:  []string{},
			obj:         generateFakeObjMeta([]string{"test.io/test1"}, nil),
			expectedObj: generateFakeObjMeta([]string{"test.io/test1"}, nil),
		},
		{
			name:        "if object is already having finalizer, finalizer should not be added",
			finalizers:  []string{"test.io/test2", "test.io/test3"},
			obj:         generateFakeObjMeta([]string{"test.io/test2", "test.io/test3"}, nil),
			expectedObj: generateFakeObjMeta([]string{"test.io/test2", "test.io/test3"}, nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "objMeta should not be nil")
			AddFinalizers(test.obj, test.finalizers)
			assert.Equal(t, test.expectedObj, test.obj, "objMeta should match")
		})
	}
}

func TestAddAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		obj         *metav1.ObjectMeta
		annotations map[string]string
		expectedObj *metav1.ObjectMeta
	}{
		{
			name:        "given annotations should be added",
			annotations: map[string]string{"test.io/key1": "val1"},
			obj:         generateFakeObjMeta(nil, nil),
			expectedObj: generateFakeObjMeta(nil, map[string]string{"test.io/key1": "val1"}),
		},
		{
			name:        "if annotation list is empty, obj should not be modified",
			annotations: nil,
			obj:         generateFakeObjMeta(nil, map[string]string{"test.io/key2": "val2"}),
			expectedObj: generateFakeObjMeta(nil, map[string]string{"test.io/key2": "val2"}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "objMeta should not be nil")
			AddAnnotations(test.obj, test.annotations)
			assert.Equal(t, test.expectedObj, test.obj, "objMeta should match")
		})
	}
}

func TestRemoveFinalizers(t *testing.T) {
	tests := []struct {
		name        string
		obj         *metav1.ObjectMeta
		finalizers  []string
		expectedObj *metav1.ObjectMeta
	}{
		{
			name:        "given finalizer should be removed",
			finalizers:  []string{"test.io/test"},
			obj:         generateFakeObjMeta([]string{"test.io/test"}, nil),
			expectedObj: generateFakeObjMeta([]string{}, nil),
		},
		{
			name:        "if finalizer list is empty, obj should not be modified",
			finalizers:  []string{},
			obj:         generateFakeObjMeta([]string{"test.io/test1"}, nil),
			expectedObj: generateFakeObjMeta([]string{"test.io/test1"}, nil),
		},
		{
			name:        "if object is not having given finalizer, obj should not be modified",
			finalizers:  []string{"test.io/test2", "test.io/test3"},
			obj:         generateFakeObjMeta([]string{"test.io/test0", "test.io/test1"}, nil),
			expectedObj: generateFakeObjMeta([]string{"test.io/test0", "test.io/test1"}, nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "objMeta should not be nil")
			RemoveFinalizers(test.obj, test.finalizers)
			assert.Equal(t, test.expectedObj, test.obj, "objMeta should match")
		})
	}
}

func TestRemoveAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		obj         *metav1.ObjectMeta
		annotations map[string]string
		expectedObj *metav1.ObjectMeta
	}{
		{
			name:        "given annotations should be removed",
			annotations: map[string]string{"test.io/key1": "val1"},
			obj:         generateFakeObjMeta(nil, map[string]string{"test.io/key1": "val1", "test.io/key2": "val2"}),
			expectedObj: generateFakeObjMeta(nil, map[string]string{"test.io/key2": "val2"}),
		},
		{
			name:        "if annotations list is empty, obj should not be modified",
			annotations: nil,
			obj:         generateFakeObjMeta(nil, map[string]string{"test.io/key1": "val1"}),
			expectedObj: generateFakeObjMeta(nil, map[string]string{"test.io/key1": "val1"}),
		},
		{
			name:        "if object is not having given annotations, obj should not be modified",
			annotations: map[string]string{"test.io/key1": "val1"},
			obj:         generateFakeObjMeta(nil, map[string]string{"test.io/key2": "val2", "test.io/key3": "val3"}),
			expectedObj: generateFakeObjMeta(nil, map[string]string{"test.io/key2": "val2", "test.io/key3": "val3"}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "objMeta should not be nil")
			RemoveAnnotations(test.obj, test.annotations)
			assert.Equal(t, test.expectedObj, test.obj, "objMeta should match")
		})
	}
}

func TestGetPatchData(t *testing.T) {
	tests := []struct {
		name           string
		oldObj         interface{}
		newObj         interface{}
		expectedString string
		expectedError  error
	}{
		{
			name:           "when both objects are same, no patch required",
			oldObj:         generateFakeServiceObj("ns1", "name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			newObj:         generateFakeServiceObj("ns1", "name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedString: "{}",
			expectedError:  nil,
		},
		{
			name:           "when both objects are not same, patch data should be returned",
			oldObj:         generateFakeServiceObj("ns1", "name1", nil, []string{"test.io/finalizer"}),
			newObj:         generateFakeServiceObj("ns1", "name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedString: "{\"metadata\":{\"annotations\":{\"test.io/key\":\"val\"}}}",
			expectedError:  nil,
		},
		{
			name:           "when object is invalid, error should be returned",
			oldObj:         "{'ns1', 'name1'}",
			newObj:         nil,
			expectedString: "{}",
			expectedError:  errors.Errorf("CreateTwoWayMergePatch failed: expected a struct, but received a string"),
		},
		{
			name:           "when old object is invalid, error should be returned",
			oldObj:         chan int(nil),
			newObj:         nil,
			expectedString: "{}",
			expectedError:  errors.Errorf("marshal old object failed: json: unsupported type: chan int"),
		},
		{
			name:           "when new object is invalid, error should be returned",
			oldObj:         nil,
			newObj:         chan int(nil),
			expectedString: "{}",
			expectedError:  errors.Errorf("marshal new object failed: json: unsupported type: chan int"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			patchBytes, _, err := GetPatchData(test.oldObj, test.newObj)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError.Error(), err.Error(), "error should match")
			} else {
				assert.Nil(t, err, "getPatchData returned error=%v", err)
				assert.Equal(t, test.expectedString, string(patchBytes), "patchData should match")
			}
		})
	}
}
