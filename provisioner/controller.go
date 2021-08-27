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
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	mKube "github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/client"
	pvController "sigs.k8s.io/sig-storage-lib-external-provisioner/v7/controller"
)

var (
	provisionerName = "openebs.io/nfsrwx"
	// LeaderElectionKey represents ENV for disable/enable leaderElection for
	// nfs provisioner
	LeaderElectionKey = "LEADER_ELECTION_ENABLED"
)

// Start will initialize and run the dynamic provisioner daemon
func Start() error {
	klog.Infof("Starting Provisioner...")

	// Dynamic Provisioner can run successfully if it can establish
	// connection to the Kubernetes Cluster. mKube helps with
	// establishing the connection either via InCluster or
	// OutOfCluster by using the following ENV variables:
	//   OPENEBS_IO_K8S_MASTER - Kubernetes master IP address
	//   OPENEBS_IO_KUBE_CONFIG - Path to the kubeConfig file.
	kubeClient, err := mKube.New().Clientset()
	if err != nil {
		return errors.Wrap(err, "unable to get k8s client")
	}

	// serverVersion, err := kubeClient.Discovery().ServerVersion()
	// if err != nil {
	// 	return errors.Wrap(err, "Cannot start Provisioner: failed to get Kubernetes server version")
	// }

	err = performPreupgradeTasks(kubeClient)
	if err != nil {
		return errors.Wrap(err, "failure in preupgrade tasks")
	}

	//Create a channel to receive shutdown signal to help
	// with graceful exit of the provisioner.
	stopCh := make(chan struct{})
	RegisterShutdownChannel(stopCh)

	//Create an instance of ProvisionerHandler to handle PV
	// create and delete events.
	provisioner, err := NewProvisioner(stopCh, kubeClient)
	if err != nil {
		return err
	}

	//Create an instance of the Dynamic Provisioner Controller
	// that has the reconciliation loops for PVC create and delete
	// events and invokes the Provisioner Handler.
	pc := pvController.NewProvisionController(
		kubeClient,
		provisionerName,
		provisioner,
		pvController.LeaderElection(isLeaderElectionEnabled()),
	)
	klog.V(4).Info("Provisioner started")

	// Create a context which can be cancled
	ctx, cancelFn := context.WithCancel(context.TODO())

	//Run the provisioner till a shutdown signal is received.
	go pc.Run(ctx)

	<-stopCh
	cancelFn()
	klog.V(4).Info("Provisioner stopped")

	return nil
}

// isLeaderElectionEnabled returns true/false based on the ENV
// LEADER_ELECTION_ENABLED set via provisioner deployment.
// Defaults to true, means leaderElection enabled by default.
func isLeaderElectionEnabled() bool {
	leaderElection := os.Getenv(LeaderElectionKey)

	var leader bool
	switch strings.ToLower(leaderElection) {
	default:
		klog.Info("Leader election enabled for nfs-provisioner")
		leader = true
	case "y", "yes", "true":
		klog.Info("Leader election enabled for nfs-provisioner via leaderElectionKey")
		leader = true
	case "n", "no", "false":
		klog.Info("Leader election disabled for nfs-provisioner via leaderElectionKey")
		leader = false
	}
	return leader
}
