package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cluster

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster customize baetyl node resource definition.
// Each cluster corresponds to an edge node or edge cluster.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec `json:"spec,omitempty"`
}

type ClusterSpec struct {
	CoreSpec     `json:",inline"`
	UserSpec     `json:",inline"`
	TemplateRef  *corev1.ObjectReference `json:"templateRef,omitempty"`
	ApplyRef     *corev1.ObjectReference `json:"applyRef,omitempty"`
	Attributes   map[string]string       `json:"attributes,omitempty"`
	OptionalApps []string                `json:"optionalApp,omitempty"`
	Accelerator  string                  `json:"accelerator,omitempty"`
	IsCluster    bool                    `json:"isCluster,omitempty"`
	SyncMode     string                  `json:"syncMode,omitempty"`
	NodeMode     string                  `json:"nodeMode,omitempty"`
	Description  string                  `json:"description,omitempty"`
}

type CoreSpec struct {
	CoreFrequency int64  `json:"coreFrequency,omitempty"`
	CoreAPIPort   int    `json:"coreAPIPort,omitempty"`
	CoreVersion   string `json:"coreVersion,omitempty"`
}

type UserSpec struct {
	UserName         string `json:"userName,omitempty"`
	UserId           string `json:"userId,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	OrganizationId   string `json:"organizationId,omitempty"`
	ProjectName      string `json:"projectName,omitempty"`
	ProjectId        string `json:"projectId,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

// Template

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Template customize baetyl template resource definition.
type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TemplateSpec `json:"spec,omitempty"`
}

type TemplateSpec struct {
	UserSpec    `json:",inline"`
	ClusterRef  *corev1.LocalObjectReference `json:"clusterRef,omitempty"`
	ApplyRef    *corev1.LocalObjectReference `json:"applyRef,omitempty"`
	Data        map[string]string            `json:"data,omitempty"`
	Description string                       `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}

// Apply

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Apply customize baetyl apply resource definition.
type Apply struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplySpec `json:"spec,omitempty"`
}

type ApplySpec struct {
	UserSpec     `json:",inline"`
	ClusterRef   *corev1.LocalObjectReference `json:"clusterRef,omitempty"`
	TemplatesRef *corev1.LocalObjectReference `json:"templatesRef,omitempty"`
	ApplyValues  []ApplyValues                `json:"applyValues,omitempty"`
	Description  string                       `json:"description,omitempty"`
}

type ApplyValues struct {
	Name        string            `json:"name,omitempty"`
	Values      map[string]string `json:"values,omitempty"`
	ExpectTime  int64             `json:"expectTime,omitempty"`
	Description string            `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ApplyList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Apply `json:"items"`
}
