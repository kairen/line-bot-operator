package v1alpha1

import (
	"github.com/line/line-bot-sdk-go/linebot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Bot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   BotSpec   `json:"spec"`
	Status BotStatus `json:"status,omitempty"`
}

type BotExposeType string

const (
	NgrokExpose        BotExposeType = "Ngrok"
	IngressExpose      BotExposeType = "Ingress"
	LoadBalancerExpose BotExposeType = "LoadBalancer"
)

type BotExpose struct {
	Type           BotExposeType `json:"type"`
	DomainName     string        `json:"domainName"`
	LoadBalanceIPs []string      `json:"loadBalanceIPs,omitempty"`
	NgrokToken     string        `json:"ngrokToken"`
}

type BotSpec struct {
	Selector          *metav1.LabelSelector `json:"selector"`
	ChannelSecretName string                `json:"channelSecretName"`
	Expose            BotExpose             `json:"expose"`
	Version           string                `json:"version"`
	LogLevel          int                   `json:"logLevel"`
}

type BotPhase string

const (
	BotPending     BotPhase = "Pending"
	BotActive      BotPhase = "Active"
	BotFailed      BotPhase = "Failed"
	BotTerminating BotPhase = "Terminating"
)

type BotStatus struct {
	Phase          BotPhase    `json:"phase"`
	Reason         string      `json:"reason,omitempty"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Bot `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec EventSpec `json:"spec"`
}

type Message struct {
	Type     linebot.MessageType `json:"type"`
	Keywords []string            `json:"keywords,omitempty"`
	Reply    string              `json:"reply"`
}

type EventSpec struct {
	Selector *metav1.LabelSelector `json:"selector"`
	Type     linebot.EventType     `json:"type"`
	Messages []Message             `json:"messages"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Event `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Subsets []EventBindingSubset `json:"subsets,omitempty"`
}

type Binding struct {
	Name     string            `json:"name"`
	Type     linebot.EventType `json:"type"`
	Messages []Message         `json:"messages"`
}

type EventBindingSubset struct {
	Binding Binding `json:"binding,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EventBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EventBinding `json:"items"`
}
