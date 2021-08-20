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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateFakeObjMeta(finalizer []string, annotations map[string]string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Finalizers:  finalizer,
		Annotations: annotations,
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
			addFinalizers(test.obj, test.finalizers)
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
			addAnnotations(test.obj, test.annotations)
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
			removeFinalizers(test.obj, test.finalizers)
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
			removeAnnotations(test.obj, test.annotations)
			assert.Equal(t, test.expectedObj, test.obj, "objMeta should match")
		})
	}
}
