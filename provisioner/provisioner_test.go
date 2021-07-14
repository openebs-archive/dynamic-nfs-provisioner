package provisioner

import (
	"os"
	"testing"

	"github.com/ghodss/yaml"
	mayav1alpha1 "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	mconfig "github.com/openebs/maya/pkg/apis/openebs.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	listersv1 "k8s.io/client-go/listers/core/v1"

	pvController "sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

// TODO: Remove below test cases

type fixture struct {
	t *testing.T

	// provisioner holds information about clientsets
	// and init configuration
	provisioner *Provisioner
}

// fixtureConfig is required create new instance of fixture
type fixtureConfig struct {
	t *testing.T

	// fake namespace where NFS server needs to run
	serverNamespace string

	// default configuration required to provsion a volume
	defaultConfig []mconfig.Config

	nodeObjects []*corev1.Node
}

func newFixture(fConfig fixtureConfig) *fixture {
	f := &fixture{
		t: fConfig.t,
		provisioner: &Provisioner{
			stopCh:          make(chan struct{}),
			kubeClient:      fake.NewSimpleClientset(),
			namespace:       fConfig.serverNamespace,
			serverNamespace: fConfig.serverNamespace,
			useClusterIP:    false,
			defaultConfig:   fConfig.defaultConfig,
		},
	}
	f.provisioner.getVolumeConfig = f.provisioner.GetVolumeConfig

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(f.provisioner.kubeClient, 0)
	k8sNodeInformer := kubeInformerFactory.Core().V1().Nodes().Informer()

	for _, nodeObj := range fConfig.nodeObjects {
		k8sNodeInformer.GetIndexer().Add(nodeObj)
	}
	f.provisioner.k8sNodeLister = listersv1.NewNodeLister(k8sNodeInformer.GetIndexer())

	return f
}

func (f *fixture) createPVC(pvcObj *corev1.PersistentVolumeClaim) func() error {
	return func() error {
		_, err := f.provisioner.kubeClient.CoreV1().
			PersistentVolumeClaims(pvcObj.Namespace).
			Create(pvcObj)
		return err
	}
}

func (f *fixture) marshal(obj interface{}) string {
	casObj, err := yaml.Marshal(obj)
	if err != nil {
		f.t.Errorf("Failed to convert object{%v} into string", obj)
		return ""
	}
	return string(casObj)
}

func (f *fixture) getFakeSCObject(scName string, casConfig []mayav1alpha1.Config) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
			Annotations: map[string]string{
				string(mayav1alpha1.CASTypeKey):   "nfsrwx",
				string(mayav1alpha1.CASConfigKey): f.marshal(casConfig),
			},
		},
		Provisioner: "openebs.io/nfsrwx",
		ReclaimPolicy: func(policy corev1.PersistentVolumeReclaimPolicy) *v1.PersistentVolumeReclaimPolicy {
			return &policy
		}(corev1.PersistentVolumeReclaimDelete),
	}
}

func TestProvision(t *testing.T) {
	fixture := newFixture(fixtureConfig{
		t:               t,
		serverNamespace: "openebs",
		defaultConfig: []mconfig.Config{
			{
				Name:  KeyPVNFSServerType,
				Value: getDefaultNFSServerType(),
			},
		},
		nodeObjects: []*corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
			},
		},
	})
	os.Setenv(string(NFSServerImageKey), "openebs/nfs-server:ci")

	tests := map[string]struct {
		options         pvController.ProvisionOptions
		createFnList    []func() error
		isErrorExpected bool
	}{
		"provision a volume without any pre-creation of objects": {
			options: pvController.ProvisionOptions{
				StorageClass: fixture.getFakeSCObject("test1-sc", []mayav1alpha1.Config{
					{
						Name:  KeyPVNFSServerType,
						Value: "kernel",
					},
					{
						Name:  KeyPVBackendStorageClass,
						Value: "backendsc-1",
					},
				}),
				PVName: "test1-pv",
				PVC:    getFakePVCObject(fixture.provisioner.serverNamespace, "test-pvc1", "test1-sc"),
			},
		},
		"provision a volume by pre-creating backend PVC": {
			options: pvController.ProvisionOptions{
				StorageClass: fixture.getFakeSCObject("test2-sc", []mayav1alpha1.Config{
					{
						Name:  KeyPVNFSServerType,
						Value: "kernel",
					},
					{
						Name:  KeyPVBackendStorageClass,
						Value: "backendsc-2",
					},
				}),
				PVName: "test2-pv",
				PVC:    getFakePVCObject(fixture.provisioner.serverNamespace, "test2-pvc", "test2-sc"),
			},
			createFnList: []func() error{
				fixture.createPVC(getFakePVCObject(fixture.provisioner.serverNamespace, "nfs-test2-pv", "test2-sc")),
			},
		},
		"provision a volume by pre-creating backend PVC & deployment": {
			options: pvController.ProvisionOptions{
				StorageClass: fixture.getFakeSCObject("test3-sc", []mayav1alpha1.Config{
					{
						Name:  KeyPVNFSServerType,
						Value: "kernel",
					},
					{
						Name:  KeyPVBackendStorageClass,
						Value: "backendsc-3",
					},
				}),
				PVName: "test3-pv",
				PVC:    getFakePVCObject(fixture.provisioner.serverNamespace, "test3-pvc", "test3-sc"),
			},
			createFnList: []func() error{
				fixture.createPVC(getFakePVCObject(fixture.provisioner.serverNamespace, "nfs-test3-pv", "test3-sc")),
			},
		},
	}
	for name, test := range tests {
		for _, fn := range test.createFnList {
			err := fn()
			if err != nil {
				t.Errorf("failed to pre-create objects error: %v", err)
				continue
			}
		}
		_, err := fixture.provisioner.kubeClient.StorageV1().StorageClasses().Create(test.options.StorageClass)
		if err != nil {
			t.Errorf(
				"%q test failed exepected error not to occur during StorageClass creation but got error %v",
				name,
				err,
			)
		}
		_, err = fixture.provisioner.Provision(test.options)
		if test.isErrorExpected && err == nil {
			t.Errorf("%q test failed expected error to occur but got nil", name)
		}
		if !test.isErrorExpected && err != nil {
			t.Errorf("%q test failed expected error not to occur but got error: %v", name, err)
		}
	}
	os.Unsetenv(string(NFSServerImageKey))
}
