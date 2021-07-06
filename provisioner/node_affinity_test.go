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

package provisioner

import (
	"os"
	"reflect"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestGetNodeAffinityRules(t *testing.T) {
	tests := map[string]struct {
		envValue         string
		expectedAffinity NodeAffinity
	}{
		"when there is only single topology without values": {
			envValue: "kubernetes.io/storage-node:[]",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies with values": {
			envValue: "kubernetes.io/storage-node:[],kubernetes.io/zone:[zone-a,zone-b,zone-c]," +
				"kubernetes.io/region:[region-1],kubernetes.io/nfs-node:[]",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/zone",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"zone-a", "zone-b", "zone-c"},
					},
					{
						Key:      "kubernetes.io/region",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"region-1"},
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies without values": {
			envValue: "kubernetes.io/storage-node:[],kubernetes.io/nfs-node:[]",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies without values & []": {
			envValue: "kubernetes.io/storage-node ,kubernetes.io/nfs-node",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies without empty values([])": {
			envValue: "kubernetes.io/nfs-node,kubernetes.io/storage-node,kubernetes.io/zone:[zone-a,zone-b,zone-c]," +
				"kubernetes.io/region:[region-1],kubernetes.io/nfs-node",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/zone",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"zone-a", "zone-b", "zone-c"},
					},
					{
						Key:      "kubernetes.io/region",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"region-1"},
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies without values but one of them has empty": {
			envValue: "kubernetes.io/storage-node,kubernetes.io/nfs-node:[]",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
		"when there are multiple topologies with empty and without values": {
			envValue: "kubernetes.io/storage-node:[],kubernetes.io/nfs-node",
			expectedAffinity: NodeAffinity{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/storage-node",
						Operator: corev1.NodeSelectorOpExists,
					},
					{
						Key:      "kubernetes.io/nfs-node",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			},
		},
	}

	for name, test := range tests {
		os.Setenv(NODEAFFINITYKEY, test.envValue)
		gotNodeAffinityRules := getNodeAffinityRules()
		if !reflect.DeepEqual(gotNodeAffinityRules.MatchExpressions, test.expectedAffinity.MatchExpressions) {
			t.Errorf(
				"%q test got failed expected %v but got %v",
				name,
				test.expectedAffinity.MatchExpressions,
				gotNodeAffinityRules.MatchExpressions,
			)
		}

		os.Unsetenv(NODEAFFINITYKEY)
	}
}

func TestGetRightMostMatchingIndex(t *testing.T) {
	tests := map[string]struct {
		regexp         *regexp.Regexp
		str            string
		expectedString string
	}{
		"When repitative pattern exist twice": {
			regexp:         regexp.MustCompile(`,+.*:\[.*`),
			str:            "key1,key2,key3:[v1,v2,v3]",
			expectedString: "key3:[v1,v2,v3]",
		},
		"When pattern exist exactly once": {
			regexp:         regexp.MustCompile(`,+.*:\[.*`),
			str:            ",key3:[v1,v2,v3]",
			expectedString: "key3:[v1,v2,v3]",
		},
		"When pattern matches more than twice": {
			regexp:         regexp.MustCompile(`,+.*:\[.*`),
			str:            "key1,key2,key3,key4:[v1,v2]",
			expectedString: "key4:[v1,v2]",
		},
		"When pattern does not match with given string": {
			regexp:         regexp.MustCompile(`abcd`),
			str:            "openebs",
			expectedString: "",
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			gotString := getRightMostMatchingString(test.regexp, test.str)
			if gotString != test.expectedString {
				t.Errorf("%q test failed expected: %q but got %q", name, test.expectedString, gotString)
			}
		})
	}
}
