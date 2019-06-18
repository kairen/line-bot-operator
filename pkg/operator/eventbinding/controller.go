package eventbinding

import (
	"reflect"

	linev1alpha1 "github.com/kairen/line-bot-operator/pkg/apis/line/v1alpha1"
	clientset "github.com/kairen/line-bot-operator/pkg/generated/clientset/versioned"
	opkit "github.com/kubedev/operator-kit"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	customResourceName       = "eventbinding"
	customResourceNamePlural = "eventbindings"
)

var Resource = opkit.CustomResource{
	Name:    customResourceName,
	Plural:  customResourceNamePlural,
	Group:   linev1alpha1.CustomResourceGroup,
	Version: linev1alpha1.Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(linev1alpha1.EventBinding{}).Name(),
}

type Controller struct {
	ctx       *opkit.Context
	clientset clientset.Interface
}

func NewController(ctx *opkit.Context, clientset clientset.Interface) *Controller {
	return &Controller{ctx: ctx, clientset: clientset}
}

func (c *Controller) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	klog.Infof("Start watching eventbinding resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.LineV1alpha1().RESTClient())
	go watcher.Watch(&linev1alpha1.EventBinding{}, stopCh)
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	eventbind := obj.(*linev1alpha1.EventBinding).DeepCopy()
	klog.V(2).Infof("Received onAdd on EventBinding %s in %s namespace.", eventbind.Name, eventbind.Namespace)
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	new := newObj.(*linev1alpha1.EventBinding).DeepCopy()
	klog.V(2).Infof("Received onUpdate on EventBinding %s in %s namespace.", new.Name, new.Namespace)
}

func (c *Controller) onDelete(obj interface{}) {
	eventbind := obj.(*linev1alpha1.EventBinding).DeepCopy()
	klog.V(2).Infof("Received onDelete on EventBinding %s in %s namespace.", eventbind.Name, eventbind.Namespace)
}
