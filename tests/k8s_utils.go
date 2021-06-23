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

package tests

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient interface for k8s API
type KubeClient struct {
	kubernetes.Interface
}

// Client for KubeClient
var Client *KubeClient

// encoder to print object in yaml format
var encoder runtime.Encoder

// getHomeDir gets the home directory for the system.
// It is required to locate the .kube/config file
func getHomeDir() (string, error) {
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}

	return "", fmt.Errorf("not able to locate home directory")
}

// getConfigPath returns the filepath of kubeconfig file
func getConfigPath() (string, error) {
	home, err := getHomeDir()
	if err != nil {
		return "", err
	}
	kubeConfigPath := home + "/.kube/config"
	return kubeConfigPath, nil
}

func initK8sClient(kubeConfigPath string) error {
	var err error
	if kubeConfigPath == "" {
		kubeConfigPath, err = getConfigPath()
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil
	}

	Client = &KubeClient{client}

	scheme := runtime.NewScheme()
	serializerInfo, found := runtime.SerializerInfoForMediaType(serializer.NewCodecFactory(scheme).SupportedMediaTypes(), "application/yaml")
	if found {
		encoder = serializerInfo.Serializer
	}
	return nil
}

func (k *KubeClient) waitForPods(podNamespace, labelSelector string, expectedPhase corev1.PodPhase, expectedCount int) error {
	dumpLog := 0
	for {
		podList, err := k.CoreV1().Pods(podNamespace).List(metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return err
		}

		count := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase == expectedPhase {
				count++
			}
		}

		if count == expectedCount {
			break
		}

		time.Sleep(5 * time.Second)

		if dumpLog > 6 {
			fmt.Printf("checking for pod with labelSelector=%s in ns=%s, count=%d expectedCount=%d\n", labelSelector, podNamespace, count, expectedCount)
			dumpLog = 0
		}
		dumpLog++
	}
	return nil
}

func (k *KubeClient) createNamespace(namespace string) error {
	_, err := k.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			o := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = k.CoreV1().Namespaces().Create(o)
		}
	}
	return err
}

// WaitForNamespaceCleanup wait for cleanup of the given namespace
func (k *KubeClient) WaitForNamespaceCleanup(ns string) error {
	dumpLog := 0
	for {
		nsObj, err := k.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if dumpLog > 6 {
			fmt.Printf("Waiting for cleanup of namespace %s\n", ns)
			dumpK8sObject(nsObj)
			dumpLog = 0
		}

		dumpLog++
		time.Sleep(5 * time.Second)
	}
}

func (k *KubeClient) destroyNamespace(namespace string) error {
	err := k.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return k.WaitForNamespaceCleanup(namespace)
	}
	return nil
}

func (k *KubeClient) waitForPVCBound(pvc, ns string) (corev1.PersistentVolumeClaimPhase, error) {
	for {
		o, err := k.CoreV1().
			PersistentVolumeClaims(ns).
			Get(pvc, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		if o.Status.Phase == corev1.ClaimLost {
			return o.Status.Phase, errors.Errorf("PVC %s/%s in lost state", ns, pvc)
		}
		if o.Status.Phase == corev1.ClaimBound {
			return o.Status.Phase, nil
		}
		time.Sleep(5 * time.Second)
	}
}

func (k *KubeClient) createPVC(pvc *corev1.PersistentVolumeClaim) error {
	_, err := k.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	_, err = k.waitForPVCBound(pvc.Name, pvc.Namespace)
	return err
}

func (k *KubeClient) deletePVC(namespace, pvc string) error {
	err := k.CoreV1().PersistentVolumeClaims(namespace).Delete(pvc, &metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
	}

	return err
}

func (k *KubeClient) createDeployment(deployment *appsv1.Deployment) error {
	_, err := k.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return errors.Errorf("Failed to create deployment %s/%s, err=%s", deployment.Namespace, deployment.Name, err)
	}
	return nil
}

func (k *KubeClient) deleteDeployment(namespace, deployment string) error {
	return k.AppsV1().Deployments(namespace).Delete(deployment, &metav1.DeleteOptions{})
}

func dumpK8sObject(obj runtime.Object) {
	if encoder == nil {
		fmt.Printf("encoder not initilized\n")
		return
	}

	buf := new(bytes.Buffer)
	encoder.Encode(obj, buf)
	fmt.Println(string(buf.Bytes()))

}
