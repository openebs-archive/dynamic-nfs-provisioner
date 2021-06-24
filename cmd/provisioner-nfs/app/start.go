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

package app

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openebs/dynamic-nfs-provisioner/pkg/metrics"
	"github.com/openebs/dynamic-nfs-provisioner/provisioner"
	"github.com/openebs/maya/pkg/util"
)

var (
	cmdName = "provisioner"
	usage   = fmt.Sprintf("%s", cmdName)

	// defaultMetricsPath defines the path where prometheus metrics are exposed
	defaultMetricsPath = "/metrics"
	// defaultListenAddress defines the address where prometheus metrics are exposed
	defaultListenAddress = ":8085"
)

// StartProvisioner will start a new dynamic NFS provisioner
func StartProvisioner() (*cobra.Command, error) {
	var (
		metricsPath   string
		listenAddress string
	)

	// Create a new command.
	cmd := &cobra.Command{
		Use:   usage,
		Short: "Dynamic NFS Provisioner",
		Long: `Manage the NFS PVs that includes: validating, creating,
			deleting and cleanup tasks. NFS PVs are setup on top
			of other PVCs (block volumes)`,
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(Start(cmd), util.Fatal)
		},
	}

	cmd.Flags().StringVar(&metricsPath, "metrics-path", defaultMetricsPath, "path under which to expose metrics")
	cmd.Flags().StringVar(&listenAddress, "listen-address", defaultListenAddress, "address on which to expose metrics")

	// add the default command line flags as global flags to cobra command
	// flagset
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Hack: Without the following line, the logs will be prefixed with Error
	_ = flag.CommandLine.Parse([]string{})

	return cmd, nil
}

// Start will initialize and run the dynamic provisioner daemon
func Start(cmd *cobra.Command) error {
	metricPath := cmd.Flag("metrics-path").Value.String()
	metricListenAddress := cmd.Flag("listen-address").Value.String()

	prometheus.MustRegister([]prometheus.Collector{
		metrics.PersistentVolumeDeleteTotal,
		metrics.PersistentVolumeDeleteFailedTotal,
		metrics.PersistentVolumeCreateTotal,
		metrics.PersistentVolumeCreateFailedTotal,
	}...)

	go func() {
		http.Handle(metricPath, promhttp.Handler())
		fmt.Printf("Starting metric server at address [%s]", metricListenAddress)
		if err := http.ListenAndServe(metricListenAddress, nil); err != nil {
			fmt.Printf("Failed to start metric server at [%s]: %v", metricListenAddress, err)
		}
	}()

	return provisioner.Start()
}
