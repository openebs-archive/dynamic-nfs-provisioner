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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
)

// KubeClient interface for k8s API
type KubeClient struct {
	kubernetes.Interface
	config *rest.Config
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

	scheme := runtime.NewScheme()
	serializerInfo, found := runtime.SerializerInfoForMediaType(serializer.NewCodecFactory(scheme).SupportedMediaTypes(), "application/yaml")
	if found {
		encoder = serializerInfo.Serializer
	}

	Client = &KubeClient{
		Interface: client,
		config:    config,
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

func (k *KubeClient) listPods(podNamespace string, labelSelector string) (*corev1.PodList, error) {
	return k.CoreV1().Pods(podNamespace).List(metav1.ListOptions{LabelSelector: labelSelector})
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

func (k *KubeClient) getPVC(pvcNamespace, pvcName string) (*corev1.PersistentVolumeClaim, error) {
	return k.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(pvcName, metav1.GetOptions{})
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

func (k *KubeClient) applyDeployment(deployment *appsv1.Deployment) error {
	// TODO: Use server side apply
	_, err := k.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Errorf("Failed to create deployment %s/%s, err=%s", deployment.Namespace, deployment.Name, err)
	}
	if err == nil {
		return nil
	}

	currentDeployment, err := k.AppsV1().
		Deployments(deployment.Namespace).
		Get(deployment.Name, metav1.GetOptions{})

	data, _, err := getPatchData(currentDeployment, deployment)
	if err != nil {
		return err
	}

	// Patch the depployment
	_, err = k.AppsV1().
		Deployments(deployment.Namespace).
		Patch(deployment.Name,
			types.StrategicMergePatchType,
			data,
		)
	if err != nil {
		return err
	}

	for {
		deployObj, err := k.AppsV1().Deployments(deployment.Namespace).
			Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		revision, err := deploymentutil.Revision(deployObj)
		if err != nil {
			return err
		}
		statusViewer := DeploymentStatusViewer{}
		msg, rolledOut, err := statusViewer.Status(deployObj, revision)
		if err != nil {
			return err
		}
		fmt.Printf("rollout status: %s\n", msg)
		if rolledOut {
			break
		}
		time.Sleep(5 * time.Second)
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

func (k *KubeClient) createStorageClass(sc *storagev1.StorageClass) error {
	_, err := k.StorageV1().StorageClasses().Create(sc)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (k *KubeClient) deleteStorageClass(scName string) error {
	return k.StorageV1().StorageClasses().Delete(scName, &metav1.DeleteOptions{})
}

func getPatchData(oldObj, newObj interface{}) ([]byte, []byte, error) {
	oldData, err := json.Marshal(oldObj)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal old object failed: %v", err)
	}
	newData, err := json.Marshal(newObj)
	if err != nil {
		return nil, nil, fmt.Errorf("mashal new object failed: %v", err)
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, oldObj)
	if err != nil {
		return nil, nil, fmt.Errorf("CreateTwoWayMergePatch failed: %v", err)
	}
	return patchBytes, oldData, nil
}
