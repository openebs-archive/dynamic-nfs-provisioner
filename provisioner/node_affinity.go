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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// getNodeAffinityRules fetchs node affinity rules from
// environment value
func getNodeAffinityRules() NodeAffinity {
	var nodeAffinity NodeAffinity

	affinityValue := getNfsServerNodeAffinity()
	if affinityValue == "" {
		return nodeAffinity
	}

	rules := strings.Split(affinityValue, "],")
	for _, rule := range rules {
		nodeAffinity.MatchExpressions = append(nodeAffinity.MatchExpressions, getOneOrMoreNodeSelectorRequirements(rule)...)
	}

	return nodeAffinity
}

// getOneOrMoreNodeSelectorRequirements can take one or more node affinity requirements
// as string and convert them to structured form of Requirements
// Ex:
//	  Case1 - Input argument: kubernetes.io/storage-node,kubernetes.io/nfs-node,kubernetes.io/zone:[zone-1,zone-2,zone-3]
//
//	  Return value:
//		- key: kubernetes.io/storage-node
//		  operator: Exists
//		- key: kubernetes.io/nfs-node
//		  operator: Exists
//		- key: kubernetes.io/zone
//		  operator: In
//		  values:
//		  - zone-1
//		  - zone-2
//		  - zone-3
//
//    Case2 - Input argument: kubernetes.io/storage-node,kubernetes.io/nfs-node,kubernetes.io/linux-amd64
//
//    Return value:
//		- key: kubernetes.io/storage-node
//		  operator: Exists
//		- key: kubernetes.io/nfs-node
//		  operator: Exists
//		- key: kubernetes.io/linux-amd64
//		  operator: Exists
//
//    Case3 - Input argument: kubernetes.io/zone:[zone-1,zone-2]
//
//    Return value:
//		- key: kubernetes.io/zone
//		  operator: In
//		  values:
//		  - zone-1
//		  - zone-2
func getOneOrMoreNodeSelectorRequirements(
	requirementsAsValue string) []corev1.NodeSelectorRequirement {
	var nodeRequirements []corev1.NodeSelectorRequirement
	var complexReq corev1.NodeSelectorRequirement
	// isComplexRequirement will be true when input is: <key1>,<key2>,<key3>:[value1, value2]
	// NOTE: Valued key-value pair will be always at end
	isComplexRequirement := regexp.MustCompile(`.*,+.*:\[.*`).FindString(requirementsAsValue) != ""

	if isComplexRequirement {
		matchingString := getRightMostMatchingString(regexp.MustCompile(`,.*:\[.*`), requirementsAsValue)
		// If input argument is Case 1
		if matchingString != "" {
			matchingIndex := strings.LastIndex(requirementsAsValue, matchingString)
			complexReq = getNodeSelectorRequirement(requirementsAsValue[matchingIndex:])
			requirementsAsValue = requirementsAsValue[:matchingIndex]
		}
	}

	// After processing complex now we will left with two cases
	// C1: <key1>,<key2>
	// C2: <key3>:[value2] --- Original Case3
	if strings.ContainsRune(requirementsAsValue, rune('[')) {
		// Case3
		nodeRequirements = append(nodeRequirements, getNodeSelectorRequirement(requirementsAsValue))
	} else {
		// Case2
		for _, req := range strings.Split(requirementsAsValue, ",") {
			if strings.TrimSpace(req) != "" {
				nodeRequirements = append(nodeRequirements, getNodeSelectorRequirement(req))
			}
		}
		if isComplexRequirement {
			nodeRequirements = append(nodeRequirements, complexReq)
		}
	}

	return nodeRequirements
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
//
// Example: kubernetes.io/hostName:[region-1,region-2 value convert as below
//
//			key: kubernetes.io/hostName
//			operator: "In"
//			values:
//			- region-1
//			- region-2
//
// Example: kubernetes.io/storage-node
//
//			key: kubernetes.io/storage-node
//			operator: "Exists"
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

// getRightMostMatchingString will return right must matching string
// which satisfies given pattern
// Example:
//	- Fetch last pattern matching on string
//		Pattern: {,.*:\[.*} string: "key1,key2,key3:[v1, v2, v3]"
//		Return value: key3:[v1, v2, v3]
func getRightMostMatchingString(regex *regexp.Regexp, value string) string {
	loc := regex.FindStringIndex(value)
	if len(loc) == 0 {
		// given value is not satisfying regular expression
		return ""
	}
	value = value[loc[0]:]
	if value[0] == ',' && len(value) > 1 {
		value = value[1:]
	}
	rightMostMatchingString := getRightMostMatchingString(regex, value)

	// If substring matching to regular expression is found then return
	// right most index
	if rightMostMatchingString != "" {
		return rightMostMatchingString
	}
	// else return starting location
	return value
}
