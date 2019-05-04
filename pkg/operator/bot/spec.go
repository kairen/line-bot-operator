package bot

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"text/template"

	linev1alpha1 "github.com/kubedev/line-bot-operator/pkg/apis/line/v1alpha1"
	"github.com/kubedev/line-bot-operator/pkg/constants"
	"github.com/kubedev/line-bot-operator/pkg/k8sutil"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var ngrokConfigTmpl = template.Must(template.New("ngork-configTmpl").Funcs(template.FuncMap{
	"printMapInOrder": printMapInOrder,
}).Parse(`web_addr: 0.0.0.0:4040
update: false
log: stdout
authtoken: {{.Authtoken}}`))

func (c *Controller) createConfigMap(bot *linev1alpha1.Bot) error {
	opts := struct {
		Authtoken string
	}{
		Authtoken: bot.Spec.Expose.NgrokToken,
	}

	b := bytes.Buffer{}
	if err := ngrokConfigTmpl.Execute(&b, opts); err != nil {
		return err
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ngrok-%s-config", bot.Name),
			Namespace: bot.Namespace,
		},
		Data: map[string]string{
			"ngrok.yml": b.String(),
		},
	}

	k8sutil.SetOwnerRef(c.ctx.Clientset, bot.Namespace, &cm.ObjectMeta, c.makeOnwerRefer(bot))
	_, err := c.ctx.Clientset.CoreV1().ConfigMaps(bot.Namespace).Create(cm)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) createService(bot *linev1alpha1.Bot) error {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bot.Name,
			Namespace: bot.Namespace,
			Labels:    map[string]string{"bot": bot.Name},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"bot": bot.Name},
			Type:     v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Name:     "bot-http",
					Port:     int32(8080),
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "ngrok-http",
					Port:     int32(4040),
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}

	k8sutil.SetOwnerRef(c.ctx.Clientset, bot.Namespace, &svc.ObjectMeta, c.makeOnwerRefer(bot))
	_, err := c.ctx.Clientset.CoreV1().Services(bot.Namespace).Create(svc)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) createDeployment(bot *linev1alpha1.Bot) error {
	defaultMode := new(int32)
	*defaultMode = 420

	podSpec := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"bot": bot.Name},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: constants.ServiceAccountName,
			Containers: []v1.Container{
				c.makeBotContainer(bot),
				c.makeNgrokContainer(bot),
			},
			RestartPolicy: v1.RestartPolicyAlways,
			Volumes: []v1.Volume{
				v1.Volume{
					Name: "ngrok-config",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: fmt.Sprintf("ngrok-%s-config", bot.Name)},
							DefaultMode:          defaultMode,
						},
					},
				},
			},
		},
	}

	replicas := int32(1)
	d := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bot.Name,
			Namespace: bot.Namespace,
			Labels:    map[string]string{"bot": bot.Name},
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"bot": bot.Name},
			},
			Template: podSpec,
			Replicas: &replicas,
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
		},
	}

	k8sutil.SetOwnerRef(c.ctx.Clientset, bot.Namespace, &d.ObjectMeta, c.makeOnwerRefer(bot))
	_, err := c.ctx.Clientset.AppsV1().Deployments(bot.Namespace).Create(d)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) makeBotContainer(bot *linev1alpha1.Bot) v1.Container {
	namespace := bot.Namespace
	if namespace == "" {
		namespace = "default"
	}

	container := v1.Container{
		Name:  "linebot",
		Image: fmt.Sprintf("%s:%s", constants.BotImageName, bot.Spec.Version),
		Args:  []string{"--logtostderr", fmt.Sprintf("--v=%d", bot.Spec.LogLevel)},
		Env: []v1.EnvVar{
			v1.EnvVar{
				Name: "CHANNEL_SECRET",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: bot.Spec.ChannelSecretName},
						Key:                  "channelSecret",
					},
				},
			},
			v1.EnvVar{
				Name: "CHANNEL_TOKEN",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: bot.Spec.ChannelSecretName},
						Key:                  "channelToken",
					},
				},
			},
			v1.EnvVar{
				Name:  "EVENTBINDING_NAME",
				Value: bot.Name,
			},
			v1.EnvVar{
				Name:  "BASE_URL_NAME",
				Value: "callback",
			},
			v1.EnvVar{
				Name:  "NAMESPACE",
				Value: namespace,
			},
		},
		Ports: []v1.ContainerPort{
			{
				Name:          "bot-http",
				ContainerPort: int32(8080),
				Protocol:      v1.ProtocolTCP,
			},
		},
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/healthz",
					Port:   intstr.FromInt(8080),
					Scheme: v1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
			SuccessThreshold:    1,
			TimeoutSeconds:      3,
		},
	}
	return container
}

func (c *Controller) makeNgrokContainer(bot *linev1alpha1.Bot) v1.Container {
	container := v1.Container{
		Name:  "ngrok",
		Image: fmt.Sprintf("%s:%s", constants.NgrokImageName, bot.Spec.Version),
		Command: []string{
			"./ngrok",
			"http",
			"8080",
		},
		Ports: []v1.ContainerPort{
			{
				Name:          "ngrok-http",
				ContainerPort: int32(4040),
				Protocol:      v1.ProtocolTCP,
			},
		},
		LivenessProbe: &v1.Probe{
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/api/tunnels",
					Port:   intstr.FromInt(4040),
					Scheme: v1.URISchemeHTTP,
				},
			},
			FailureThreshold:    3,
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
			SuccessThreshold:    1,
			TimeoutSeconds:      3,
		},
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      "ngrok-config",
				MountPath: "/home/ngrok/.ngrok2",
			},
		},
	}
	return container
}

func (c *Controller) createEventBinding(bot *linev1alpha1.Bot) error {
	eb := &linev1alpha1.EventBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bot.Name,
			Namespace: bot.Namespace,
			Labels:    bot.Spec.Selector.MatchLabels,
		},
	}

	k8sutil.SetOwnerRef(c.ctx.Clientset, bot.Namespace, &eb.ObjectMeta, c.makeOnwerRefer(bot))
	_, err := c.clientset.LineV1alpha1().EventBindings(bot.Namespace).Create(eb)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) makeOnwerRefer(bot *linev1alpha1.Bot) *metav1.OwnerReference {
	return metav1.NewControllerRef(bot, schema.GroupVersionKind{
		Group:   linev1alpha1.SchemeGroupVersion.Group,
		Version: linev1alpha1.SchemeGroupVersion.Version,
		Kind:    reflect.TypeOf(linev1alpha1.Bot{}).Name(),
	})
}

func printMapInOrder(m map[string]string, sep string) []string {
	if m == nil {
		return nil
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		keys[i] = fmt.Sprintf("%s%s\"%s\"", k, sep, m[k])
	}
	return keys
}
