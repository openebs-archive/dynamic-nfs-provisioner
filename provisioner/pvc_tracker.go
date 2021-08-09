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
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
)

// ProvisioningTracker tracks provisioning request
type ProvisioningTracker interface {
	// Add PV for which provisioning is in-progress
	Add(pvName string)

	// Delete PV for which provisioning is completed
	Delete(pvName string)

	// Inprogress checks if provisioning for given PV is in-progress or not
	Inprogress(pvName string) bool
}

type provisioningTracker struct {
	// request contains list of in-progress provisioning request
	request sets.String
	lock    sync.RWMutex
}

func NewProvisioningTracker() ProvisioningTracker {
	return &provisioningTracker{
		request: sets.NewString(),
	}
}

func (t *provisioningTracker) Add(pvName string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.request.Insert(pvName)
}

func (t *provisioningTracker) Delete(pvName string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.request.Delete(pvName)
}

func (t *provisioningTracker) Inprogress(pvName string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.request.Has(pvName)
}
