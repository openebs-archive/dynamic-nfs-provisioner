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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/reference"
)

// KubeClient interface for k8s API
type KubeClient struct {
	kubernetes.Interface
	config *rest.Config
}

var (
	// Client for KubeClient
	Client *KubeClient

	// encoder to print object in yaml format
	encoder runtime.Encoder

	// defaultChunkSize is a maximum number of responses to
	// return for a list call. If still resources exist then
	// server will set continue field in listOptions so it is
	// responsibility of client to fetch further responses if
	// continue field is set
	defaultChunkSize = int64(500)
	metadataAccessor = meta.NewAccessor()
)

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
		if err != nil {
			return err
		}
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
		podList, err := k.CoreV1().Pods(podNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
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
	return k.CoreV1().Pods(podNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
}

// Lists Pods created from a Deployment
func (k *KubeClient) listDeploymentPods(deploy *appsv1.Deployment) (*corev1.PodList, error) {
	if deploy == nil {
		return nil, errors.Errorf("failed to get PodList: invalid input")
	}

	var labelSelector string
	for key, val := range deploy.Spec.Selector.MatchLabels {
		labelSelector += key + "=" + val + ","
	}
	labelSelector = strings.TrimSuffix(labelSelector, ",")

	return k.listPods(deploy.Namespace, labelSelector)
}

func (k *KubeClient) createNamespace(namespace string) error {
	_, err := k.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			o := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = k.CoreV1().Namespaces().Create(context.TODO(), o, metav1.CreateOptions{})
		}
	}
	return err
}

// WaitForNamespaceCleanup wait for cleanup of the given namespace
func (k *KubeClient) WaitForNamespaceCleanup(ns string) error {
	dumpLog := 0
	for {
		nsObj, err := k.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
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
	err := k.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return k.WaitForNamespaceCleanup(namespace)
	}
	return nil
}

func (k *KubeClient) waitForPVCBound(ns, pvcName string) (corev1.PersistentVolumeClaimPhase, error) {
	for {
		o, err := k.CoreV1().
			PersistentVolumeClaims(ns).
			Get(context.TODO(), pvcName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		if o.Status.Phase == corev1.ClaimLost {
			return o.Status.Phase, errors.Errorf("PVC %s/%s in lost state", ns, pvcName)
		}
		if o.Status.Phase == corev1.ClaimBound {
			return o.Status.Phase, nil
		}
		fmt.Printf("waiting for PVC {%s} in namespace {%s} to get into bound state\n", pvcName, ns)
		time.Sleep(5 * time.Second)
	}
}

// createPVC will create PVC and it will not wait for PVC to get bound
func (k *KubeClient) createPVC(pvc *corev1.PersistentVolumeClaim) error {
	_, err := k.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func (k *KubeClient) getPVC(pvcNamespace, pvcName string) (*corev1.PersistentVolumeClaim, error) {
	return k.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
}

func (k *KubeClient) deletePVC(namespace, pvc string) error {
	err := k.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvc, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
	}

	return err
}

func (k *KubeClient) getPV(name string) (*corev1.PersistentVolume, error) {
	return k.CoreV1().PersistentVolumes().Get(context.TODO(), name, metav1.GetOptions{})
}

func (k *KubeClient) updatePV(pvObj *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	return k.CoreV1().PersistentVolumes().Update(context.TODO(), pvObj, metav1.UpdateOptions{})
}

func (k *KubeClient) deletePV(pvName string) error {
	return k.CoreV1().PersistentVolumes().Delete(context.TODO(), pvName, metav1.DeleteOptions{})
}

func (k *KubeClient) createDeployment(deployment *appsv1.Deployment) error {
	_, err := k.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
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
	currentDeployment, err := k.AppsV1().
		Deployments(deployment.Namespace).
		Get(context.TODO(), deployment.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			_, err := k.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
			if err != nil {
				return errors.Errorf("Failed to create deployment %s/%s, err=%s", deployment.Namespace, deployment.Name, err)
			}
		}
		return err
	}

	data, _, err := getPatchData(currentDeployment, deployment)
	if err != nil {
		return err
	}

	// Patch the deployment
	_, err = k.AppsV1().
		Deployments(deployment.Namespace).
		Patch(context.TODO(), deployment.Name,
			types.StrategicMergePatchType,
			data,
			metav1.PatchOptions{},
		)
	if err != nil {
		return err
	}

	return k.waitForDeploymentRollout(deployment.Namespace, deployment.Name)
}

func (k *KubeClient) deleteDeployment(namespace, deployment string) error {
	return k.AppsV1().Deployments(namespace).Delete(context.TODO(), deployment, metav1.DeleteOptions{})
}

func (k *KubeClient) getDeployment(namespace, deployment string) (*appsv1.Deployment, error) {
	return k.AppsV1().Deployments(namespace).Get(context.TODO(), deployment, metav1.GetOptions{})
}

func (k *KubeClient) updateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return k.AppsV1().Deployments(deployment.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
}

func (k *KubeClient) listDeployments(namespace, labelSelector string) (*appsv1.DeploymentList, error) {
	return k.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
}

func dumpK8sObject(obj runtime.Object) {
	if encoder == nil {
		fmt.Printf("encoder not initialized\n")
		return
	}

	buf := new(bytes.Buffer)
	encoder.Encode(obj, buf)
	fmt.Println(string(buf.Bytes()))
}

func (k *KubeClient) createStorageClass(sc *storagev1.StorageClass) error {
	_, err := k.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (k *KubeClient) deleteStorageClass(scName string) error {
	return k.StorageV1().StorageClasses().Delete(context.TODO(), scName, metav1.DeleteOptions{})
}

// Add Kubernetes service related operations
func (k *KubeClient) getService(namespace, name string) (*corev1.Service, error) {
	return k.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (k *KubeClient) deleteService(namespace, name string) error {
	return k.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// Add Node related operations
func (k *KubeClient) listNodes(labelSelector string) (*corev1.NodeList, error) {
	return k.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
}

func (k *KubeClient) updateNode(node *corev1.Node) (*corev1.Node, error) {
	return k.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
}

func (k *KubeClient) getEvents(objOrRef runtime.Object) (*corev1.EventList, error) {
	ref, err := reference.GetReference(scheme.Scheme, objOrRef)
	if err != nil {
		return nil, err
	}
	stringRefKind := string(ref.Kind)
	var refKind *string
	if len(stringRefKind) > 0 {
		refKind = &stringRefKind
	}
	stringRefUID := string(ref.UID)
	var refUID *string
	if len(stringRefUID) > 0 {
		refUID = &stringRefUID
	}

	e := k.CoreV1().Events(ref.Namespace)
	fieldSelector := e.GetFieldSelector(&ref.Name, &ref.Namespace, refKind, refUID)
	initialOpts := metav1.ListOptions{FieldSelector: fieldSelector.String(), Limit: defaultChunkSize}
	eventList := &corev1.EventList{}
	err = followContinue(&initialOpts,
		func(options metav1.ListOptions) (runtime.Object, error) {
			newEvents, err := e.List(context.TODO(), options)
			if err != nil {
				return nil, err
			}
			eventList.Items = append(eventList.Items, newEvents.Items...)
			return newEvents, nil
		})
	return eventList, err
}

// followContinue handles the continue parameter returned
// by the API server when using list chunking. To take
// advantage of this, the initial ListOptions provided by
// the consumer should include a non-zero Limit parameter.
func followContinue(initialOpts *metav1.ListOptions,
	listFunc func(metav1.ListOptions) (runtime.Object, error)) error {
	opts := initialOpts
	for {
		list, err := listFunc(*opts)
		if err != nil {
			return err
		}
		nextContinueToken, _ := metadataAccessor.Continue(list)
		if len(nextContinueToken) == 0 {
			return nil
		}
		opts.Continue = nextContinueToken
	}
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

func (k *KubeClient) waitForDeploymentRollout(ns, deployment string) error {
	return wait.PollInfinite(2*time.Second, func() (bool, error) {
		deploy, err := k.AppsV1().Deployments(ns).Get(context.TODO(), deployment, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		var cond *appsv1.DeploymentCondition
		// list all conditions and and select that condition which type is Progressing.
		for i := range deploy.Status.Conditions {
			c := deploy.Status.Conditions[i]
			if c.Type == appsv1.DeploymentProgressing {
				cond = &c
			}
		}
		// if deploy.Generation <= deploy.Status.ObservedGeneration then deployment spec is not updated yet.
		// it marked IsRolledout as false and update message accordingly
		if deploy.Generation <= deploy.Status.ObservedGeneration {
			// If Progressing condition's reason is ProgressDeadlineExceeded then it is not rolled out.
			if cond != nil && cond.Reason == "ProgressDeadlineExceeded" {
				return false, errors.New(fmt.Sprintf("deployment exceeded its progress deadline"))
			}
			// if deploy.Status.UpdatedReplicas < *deploy.Spec.Replicas then some of the replicas are updated
			// and some of them are not. It marked IsRolledout as false and update message accordingly
			if deploy.Spec.Replicas != nil && deploy.Status.UpdatedReplicas < *deploy.Spec.Replicas {
				fmt.Printf("Waiting for deployment rollout to finish: %d out of %d new replicas have been updated\n",
					deploy.Status.UpdatedReplicas, *deploy.Spec.Replicas)
				return false, nil
			}
			// if deploy.Status.Replicas > deploy.Status.UpdatedReplicas then some of the older replicas are in running state
			// because newer replicas are not in running state. It waits for newer replica to come into running state then terminate.
			// It marked IsRolledout as false and update message accordingly
			if deploy.Status.Replicas > deploy.Status.UpdatedReplicas {
				fmt.Printf("Waiting for deployment rollout to finish: %d old replicas are pending termination\n",
					deploy.Status.Replicas-deploy.Status.UpdatedReplicas)
				return false, nil
			}
			// if deploy.Status.AvailableReplicas < deploy.Status.UpdatedReplicas then all the replicas are updated but they are
			// not in running state. It marked IsRolledout as false and update message accordingly.
			if deploy.Status.AvailableReplicas < deploy.Status.UpdatedReplicas {
				fmt.Printf("Waiting for deployment rollout to finish: %d of %d updated replicas are available\n",
					deploy.Status.AvailableReplicas, deploy.Status.UpdatedReplicas)
			}
			return true, nil
		}
		fmt.Printf("Waiting for deployment spec update to be observed\n")
		return false, nil
	})
}

func (k *KubeClient) listEvents(namespace string) (*corev1.EventList, error) {
	return k.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
}

// createConfigMap will create k8s resource for given configMap object
func (k *KubeClient) createConfigMap(cmap *corev1.ConfigMap) error {
	_, err := k.CoreV1().ConfigMaps(cmap.Namespace).Create(context.TODO(), cmap, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (k *KubeClient) deleteConfigMap(namespace, configMapName string) error {
	err := k.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), configMapName, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
	}

	return err
}
