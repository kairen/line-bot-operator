package bot

import (
	"reflect"
	"time"

	opkit "github.com/kubedev/operator-kit"
	linev1alpha1 "github.com/kubedev/line-bot-operator/pkg/apis/line/v1alpha1"
	clientset "github.com/kubedev/line-bot-operator/pkg/generated/clientset/versioned"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	customResourceName       = "bot"
	customResourceNamePlural = "bots"
)

var Resource = opkit.CustomResource{
	Name:    customResourceName,
	Plural:  customResourceNamePlural,
	Group:   linev1alpha1.CustomResourceGroup,
	Version: linev1alpha1.Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(linev1alpha1.Bot{}).Name(),
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

	klog.Infof("Start watching bot resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.LineV1alpha1().RESTClient())
	go watcher.Watch(&linev1alpha1.Bot{}, stopCh)
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	bot := obj.(*linev1alpha1.Bot).DeepCopy()
	klog.V(2).Infof("Received onAdd on Bot %s in %s namespace.", bot.Name, bot.Namespace)

	if bot.Status.Phase == "" {
		bot.Status.Phase = linev1alpha1.BotPending
	}

	if bot.Status.Phase == linev1alpha1.BotPending || bot.Status.Phase == linev1alpha1.BotFailed {
		if err := c.createBot(bot); err != nil {
			klog.Errorf("Failed to create bot on %s in %s namespace: %+v.", bot.Name, bot.Namespace, err)
		}
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	new := newObj.(*linev1alpha1.Bot).DeepCopy()
	klog.V(2).Infof("Received onUpdate on Bot %s in %s namespace.", new.Name, new.Namespace)
}

func (c *Controller) onDelete(obj interface{}) {
	bot := obj.(*linev1alpha1.Bot).DeepCopy()
	klog.V(2).Infof("Received onDelete on Bot %s in %s namespace.", bot.Name, bot.Namespace)
}

func (c *Controller) createBot(bot *linev1alpha1.Bot) error {
	if err := c.createConfigMap(bot); err != nil {
		return err
	}
	klog.Errorf("Success to create configmap on %s in %s namespace.", bot.Name, bot.Namespace)

	if err := c.createService(bot); err != nil {
		return err
	}
	klog.Errorf("Success to create service on %s in %s namespace.", bot.Name, bot.Namespace)

	if err := c.createDeployment(bot); err != nil {
		return err
	}
	klog.Errorf("Success to create deployment on %s in %s namespace.", bot.Name, bot.Namespace)

	if err := c.createEventBinding(bot); err != nil {
		return err
	}
	klog.Errorf("Success to create eventbinding on %s in %s namespace.", bot.Name, bot.Namespace)

	bot.Status.Phase = linev1alpha1.BotActive
	bot.Status.Reason = ""
	bot.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if _, err := c.clientset.LineV1alpha1().Bots(bot.Namespace).Update(bot); err != nil {
		return err
	}
	return nil
}
