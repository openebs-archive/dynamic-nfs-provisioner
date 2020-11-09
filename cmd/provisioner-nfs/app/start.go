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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openebs/dynamic-nfs-provisioner/provisioner"
	"github.com/openebs/maya/pkg/util"
)

var (
	cmdName = "provisioner"
	usage   = fmt.Sprintf("%s", cmdName)
)

// StartProvisioner will start a new dynamic NFS provisioner
func StartProvisioner() (*cobra.Command, error) {
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

	// add the default command line flags as global flags to cobra command
	// flagset
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Hack: Without the following line, the logs will be prefixed with Error
	_ = flag.CommandLine.Parse([]string{})

	return cmd, nil
}

// Start will initialize and run the dynamic provisioner daemon
func Start(cmd *cobra.Command) error {
	return provisioner.Start()
}
