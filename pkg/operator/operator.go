package operator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientset "github.com/kubedev/line-bot-operator/pkg/generated/clientset/versioned"
	"github.com/kubedev/line-bot-operator/pkg/k8sutil"
	"github.com/kubedev/line-bot-operator/pkg/operator/bot"
	"github.com/kubedev/line-bot-operator/pkg/operator/event"
	"github.com/kubedev/line-bot-operator/pkg/operator/eventbinding"
	opkit "github.com/kubedev/operator-kit"
	v1 "k8s.io/api/core/v1"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	initRetryDelay = 10 * time.Second
	interval       = 500 * time.Millisecond
	timeout        = 60 * time.Second
)

type Operator struct {
	ctx               *opkit.Context
	resources         []opkit.CustomResource
	botController     *bot.Controller
	eventController   *event.Controller
	bindingController *eventbinding.Controller
}

func NewMainOperator() *Operator {
	return &Operator{resources: []opkit.CustomResource{
		bot.Resource,
		event.Resource,
		eventbinding.Resource,
	}}
}

func (o *Operator) Initialize(kubeconfig string) error {
	klog.V(2).Info("Initialize the operator resources.")
	ctx, lineClient, err := o.initContextAndClient(kubeconfig)
	if err != nil {
		return err
	}

	o.botController = bot.NewController(ctx, lineClient)
	o.eventController = event.NewController(ctx, lineClient)
	o.bindingController = eventbinding.NewController(ctx, lineClient)
	o.ctx = ctx
	return nil
}

func (o *Operator) initContextAndClient(kubeconfig string) (*opkit.Context, clientset.Interface, error) {
	klog.V(2).Info("Initialize the operator context and client.")

	config, err := k8sutil.GetRestConfig(kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes config. %+v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes client. %+v", err)
	}

	extensionsClient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create Kubernetes API extension clientset. %+v", err)
	}

	lineClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create line clientset. %+v", err)
	}

	ctx := &opkit.Context{
		Clientset:             client,
		APIExtensionClientset: extensionsClient,
		Interval:              interval,
		Timeout:               timeout,
	}
	return ctx, lineClient, nil
}

func (o *Operator) initResources() error {
	klog.V(2).Info("Initialize the CRD resources.")

	ctx := opkit.Context{
		Clientset:             o.ctx.Clientset,
		APIExtensionClientset: o.ctx.APIExtensionClientset,
		Interval:              interval,
		Timeout:               timeout,
	}

	if err := opkit.CreateCustomResources(ctx, o.resources); err != nil {
		return fmt.Errorf("Failed to create custom resource. %+v", err)
	}
	return nil
}

func (o *Operator) Run() error {
	for {
		err := o.initResources()
		if err == nil {
			break
		}
		klog.Errorf("Failed to init resources. %+v. retrying...", err)
		<-time.After(initRetryDelay)
	}

	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// start watching the resources
	o.bindingController.StartWatch(v1.NamespaceAll, stopChan)
	o.eventController.StartWatch(v1.NamespaceAll, stopChan)
	o.botController.StartWatch(v1.NamespaceAll, stopChan)

	for {
		select {
		case <-signalChan:
			klog.Infof("Shutdown signal received, exiting...")
			close(stopChan)
			return nil
		}
	}
}
