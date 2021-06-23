/*
Copyright 2019-2020 The OpenEBS Authors

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

package tests

import (
	"flag"
	"fmt"

	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	// auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	kubeConfigPath              string
	applicationNamespace        = "nfs-tests-ns"
	err                         error
	NFSProvisionerLabelSelector = "openebs.io/component-name=openebs-nfs-provisioner"
	OpenEBSNamespace            = "openebs"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test application deployment")
}

var _ = BeforeSuite(func() {
	flag.StringVar(&kubeConfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig to invoke kubernetes API calls")
	flag.Parse()
	if err := initK8sClient(kubeConfigPath); err != nil {
		panic(fmt.Sprintf("failed to initialize k8s client err=%s", err))
	}

	By("waiting for openebs-nfs-provisioner pod to come into running state")
	err := Client.waitForPods(string(OpenEBSNamespace), string(NFSProvisionerLabelSelector), corev1.PodRunning, 1)
	Expect(err).To(BeNil(), "while waiting for nfs deployment to be ready")

	By("building a namespace")
	err = Client.createNamespace(applicationNamespace)
	Expect(err).To(BeNil(), "while creating namespace {%s}", applicationNamespace)

})

var _ = AfterSuite(func() {
	if Client == nil {
		return
	}

	By("deleting namespace")
	err = Client.destroyNamespace(applicationNamespace)
	Expect(err).To(BeNil(), "while deleting namespace {%s}", applicationNamespace)

})
