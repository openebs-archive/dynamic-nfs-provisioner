/*
Copyright 2020 The OpenEBS Authors.

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
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var (
	nodeAffinityKey = "NODEAFFINITY"
)

// getEnv fetches the provided environment variable's value
func getEnv(envKey string) (value string) {
	return strings.TrimSpace(os.Getenv(envKey))
}

// getNodeAffinityRules fetchs node affinity rules from
// environment value
func getNodeAffinityRules() NodeAffinity {
	var nodeAffinity NodeAffinity

	affinityValue := getEnv(nodeAffinityKey)
	if affinityValue == "" {
		return nodeAffinity
	}

	rules := strings.Split(affinityValue, "],")
	nodeAffinity.MatchExpressions = make([]corev1.NodeSelectorRequirement, len(rules))
	for index, rule := range rules {
		nodeAffinity.MatchExpressions[index] = getNodeSelectorRequirement(rule)
	}

	return nodeAffinity
}

// getNodeSelectorRequirement converts requirement from plain
// string to corev1.NodeSelectorRequirement
//
// Example: kubernetes.io/hostName:[z1-host1,z2-host1,z3-host1] value convert as below
//
//			key: kubernetes.io/hostName
//			operator: "In"
//			values:
//			- z1-host1
//			- z2-host1
//			- z3-host1
func getNodeSelectorRequirement(reqAsValue string) corev1.NodeSelectorRequirement {
	var nsRequirement corev1.NodeSelectorRequirement
	keyValues := strings.Split(reqAsValue, ":")
	// Key will always exist in given ENV
	nsRequirement.Key = strings.TrimSpace(keyValues[0])
	nsRequirement.Operator = corev1.NodeSelectorOpExists

	// If there exist more than one value
	if len(keyValues) > 1 {
		valueList := strings.Split(
			strings.TrimSpace(
				strings.TrimLeft(
					strings.TrimRight(keyValues[1], "]"),
					"["),
			),
			",")

		// If user mentioned list of values
		if len(valueList) > 1 || (len(valueList) == 1 && strings.TrimSpace(valueList[0]) != "") {
			nsRequirement.Operator = corev1.NodeSelectorOpIn
			nsRequirement.Values = valueList
		}
	}

	return nsRequirement
}
