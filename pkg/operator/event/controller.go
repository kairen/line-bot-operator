package event

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	linev1alpha1 "github.com/kairen/line-bot-operator/pkg/apis/line/v1alpha1"
	clientset "github.com/kairen/line-bot-operator/pkg/generated/clientset/versioned"
	opkit "github.com/kubedev/operator-kit"
	slice "github.com/thoas/go-funk"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	customResourceName       = "event"
	customResourceNamePlural = "events"
)

var Resource = opkit.CustomResource{
	Name:    customResourceName,
	Plural:  customResourceNamePlural,
	Group:   linev1alpha1.CustomResourceGroup,
	Version: linev1alpha1.Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(linev1alpha1.Event{}).Name(),
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

	klog.Infof("Start watching event resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.LineV1alpha1().RESTClient())
	go watcher.Watch(&linev1alpha1.Event{}, stopCh)
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	event := obj.(*linev1alpha1.Event).DeepCopy()
	klog.V(2).Infof("Received onAdd on Event %s in %s namespace.", event.Name, event.Namespace)

	if event.Spec.Selector != nil {
		if err := c.updateNewToBinding(event); err != nil {
			klog.Errorf("Failed to update eventbind on %s in %s namespace: %+v.", event.Name, event.Namespace, err)
		}
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	new := newObj.(*linev1alpha1.Event).DeepCopy()
	klog.V(2).Infof("Received onUpdate on Event %s in %s namespace.", new.Name, new.Namespace)

	if new.Spec.Selector != nil {
		if err := c.updateChangeToBinding(new); err != nil {
			klog.Errorf("Failed to update eventbind on %s in %s namespace: %+v.", new.Name, new.Namespace, err)
		}
	}
}

func (c *Controller) onDelete(obj interface{}) {
	event := obj.(*linev1alpha1.Event).DeepCopy()
	klog.V(2).Infof("Received onDelete on Event %s in %s namespace.", event.Name, event.Namespace)

	if event.Spec.Selector != nil {
		if err := c.updateDeleteToBinding(event); err != nil {
			klog.Errorf("Failed to update eventbind on %s in %s namespace: %+v.", event.Name, event.Namespace, err)
		}
	}
}

func (c *Controller) getEventBindingList(event *linev1alpha1.Event) (*linev1alpha1.EventBindingList, error) {
	return c.clientset.LineV1alpha1().EventBindings(event.Namespace).List(metav1.ListOptions{
		LabelSelector: c.createKeyValuePairs(event.Spec.Selector.MatchLabels),
	})
}

func (c *Controller) updateNewToBinding(event *linev1alpha1.Event) error {
	eventBindings, err := c.getEventBindingList(event)
	if err != nil {
		return err
	}

	subset := linev1alpha1.EventBindingSubset{
		Binding: linev1alpha1.Binding{
			Name:     event.Name,
			Type:     event.Spec.Type,
			Messages: event.Spec.Messages,
		},
	}
	for _, eventBinding := range eventBindings.Items {
		if !slice.Contains(eventBinding.Subsets, subset) {
			eventBinding.Subsets = append(eventBinding.Subsets, subset)
		}

		_, err := c.clientset.LineV1alpha1().EventBindings(event.Namespace).Update(&eventBinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) updateChangeToBinding(event *linev1alpha1.Event) error {
	eventBindings, err := c.getEventBindingList(event)
	if err != nil {
		return err
	}

	subset := linev1alpha1.EventBindingSubset{
		Binding: linev1alpha1.Binding{
			Name:     event.Name,
			Type:     event.Spec.Type,
			Messages: event.Spec.Messages,
		},
	}

	for _, eventBinding := range eventBindings.Items {
		result := slice.Filter(eventBinding.Subsets, func(sb linev1alpha1.EventBindingSubset) bool {
			return sb.Binding.Name != subset.Binding.Name
		})
		eventBinding.Subsets = result.([]linev1alpha1.EventBindingSubset)
		eventBinding.Subsets = append(eventBinding.Subsets, subset)
		_, err := c.clientset.LineV1alpha1().EventBindings(event.Namespace).Update(&eventBinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) updateDeleteToBinding(event *linev1alpha1.Event) error {
	eventBindings, err := c.getEventBindingList(event)
	if err != nil {
		return err
	}

	subset := linev1alpha1.EventBindingSubset{
		Binding: linev1alpha1.Binding{
			Name:     event.Name,
			Type:     event.Spec.Type,
			Messages: event.Spec.Messages,
		},
	}

	for _, eventBinding := range eventBindings.Items {
		if slice.Contains(eventBinding.Subsets, subset) {
			result := slice.Filter(eventBinding.Subsets, func(sb linev1alpha1.EventBindingSubset) bool {
				return sb.Binding.Name != subset.Binding.Name
			})
			eventBinding.Subsets = result.([]linev1alpha1.EventBindingSubset)
		}
		_, err := c.clientset.LineV1alpha1().EventBindings(event.Namespace).Update(&eventBinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=%s,", key, value)
	}
	return strings.TrimSuffix(b.String(), ",")
}
