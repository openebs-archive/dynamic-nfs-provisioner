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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// NfsProvisionerSubsystem is prometheus subsystem name.
	NfsVolumeProvisionerSubsystem = "nfs_volume_provisioner"

	// Metrics
	// ProvisionerRequestCreate represents metrics related to create resource request.
	ProvisionerRequestCreate = "create"
	// ProvisionerRequestDelete represents metrics related to delete resource request.
	ProvisionerRequestDelete = "delete"

	// Labels
	Process = "process"
)

var (
	// PersistentVolumeDeleteTotal is used to collect accumulated count of persistent volumes deleted.
	PersistentVolumeDeleteTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: NfsVolumeProvisionerSubsystem,
			Name:      "persistentvolume_delete_total",
			Help:      "Total number of persistent volumes deleted",
		},
		[]string{Process},
	)
	// PersistentVolumeDeleteFailedTotal is used to collect accumulated count of persistent volume delete failed attempts.
	PersistentVolumeDeleteFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: NfsVolumeProvisionerSubsystem,
			Name:      "persistentvolume_delete_failed_total",
			Help:      "Total number of persistent volume delete failed attempts",
		},
		[]string{Process},
	)
	// PersistentVolumeCreateTotal is used to collect accumulated count of persistent volume created.
	PersistentVolumeCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: NfsVolumeProvisionerSubsystem,
			Name:      "persistentvolume_create_total",
			Help:      "Total number of persistent volumes created",
		},
		[]string{Process},
	)
	// PersistentVolumeCreateFailedTotal is used to collect accumulated count of persistent volume create requests failed.
	PersistentVolumeCreateFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: NfsVolumeProvisionerSubsystem,
			Name:      "persistentvolume_create_failed_total",
			Help:      "Total number of persistent volume creation failed attempts",
		},
		[]string{Process},
	)
)
