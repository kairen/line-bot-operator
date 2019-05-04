package k8sutil

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

var (
	skipSetOwnerRefEnv bool
	testedSetOwnerRef  bool
)

func SetOwnerRef(clientset kubernetes.Interface, namespace string, object *metav1.ObjectMeta, ownerRef *metav1.OwnerReference) {
	if ownerRef == nil {
		return
	}
	SetOwnerRefs(clientset, namespace, object, []metav1.OwnerReference{*ownerRef})
}

func SetOwnerRefs(clientset kubernetes.Interface, namespace string, object *metav1.ObjectMeta, ownerRefs []metav1.OwnerReference) {
	if !testedSetOwnerRef {
		testSetOwnerRef(clientset, namespace, ownerRefs)
		testedSetOwnerRef = true
	}
	if skipSetOwnerRefEnv {
		return
	}

	// We want to set the owner ref unless we detect if it needs to be skipped.
	object.OwnerReferences = ownerRefs
}

func testSetOwnerRef(clientset kubernetes.Interface, namespace string, ownerRefs []metav1.OwnerReference) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-ownerref",
			Namespace:       namespace,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{},
	}
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(cm)
	if err != nil && !errors.IsAlreadyExists(err) {
		klog.Warningf("OwnerReferences will not be set on resources created by rook. failed to test that it can be set. %+v", err)
		skipSetOwnerRefEnv = true
		return
	}

	klog.Infof("Verified the ownerref can be set on resources")
	skipSetOwnerRefEnv = false
}
