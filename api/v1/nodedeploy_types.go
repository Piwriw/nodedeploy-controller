/*
Copyright 2024.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeDeploySpec defines the desired state of NodeDeploy
type NodeDeploySpec struct {
	// +kubebuilder:validation:Required
	NodeName string `json:"nodeName,omitempty"`
	// +kubebuilder:validation:Required
	NodeIP string `json:"nodeIP,omitempty"`
	// +kubebuilder:validation:Required
	NodeType NodeType `json:"nodeType,omitempty"`
	// +kubebuilder:default="22"
	// +optional
	NodePort string `json:"nodePort,omitempty"`
	// +kubebuilder:default=root
	// +kubebuilder:validation:Required
	NodeUser string `json:"nodeUser,omitempty"`
	// +kubebuilder:validation:Required
	NodePwd string `json:"nodePwd,omitempty"`
	// +optional
	HarborEndpoint string `json:"harborEndpoint,omitempty"`
	// +optional
	HarborUser string `json:"harborUser,omitempty"`
	// +optional
	HarborPwd string `json:"harborPwd,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`
	//Platform string `json:"platform,omitempty"`
	// +optional
	NodeVersion string `json:"nodeVersion,omitempty"`
	// +optional
	IsEvicted bool `json:"isEvicted,omitempty"`
	//UseSSHKey string `json:"useSSHKey,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=inactive;active
	NodeStatus NodeStatus `json:"nodeStatus,omitempty"`
	// +optional
	// +kubebuilder:default=3
	// if MaxRetry <= 0 disable retry
	MaxRetry int32 `json:"maxRetry,omitempty"`
}
type NodeType string

const (
	NodeWork     NodeType = "work"
	NodeKubeedge NodeType = "kubeedge"
)

func (s NodeType) String() string {
	return string(s)
}

type NodeStatus string

const (
	NodeInit          NodeStatus = "init"
	NodeInactive      NodeStatus = "inactive"
	NodeActive        NodeStatus = "active"
	NodeLaunching     NodeStatus = "launching"
	NodeLaunchFail    NodeStatus = "launchFail"
	NodeDeprecating   NodeStatus = "deprecating"
	NodeDeprecateFail NodeStatus = "deprecateFail"

	NodeUnknown NodeStatus = ""
)

func (s NodeStatus) String() string {
	if s == NodeUnknown {
		return "unknown"
	}

	return string(s)
}

// NodeDeployStatus defines the observed state of NodeDeploy
type NodeDeployStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +optional
	NodeStatus NodeStatus `json:"nodeStatus,omitempty"`
	// +optional
	// In the case of abnormal exit, the state may stay in the intermediate status
	// and a timeout mechanism is added to restore the status.
	Deadline *metav1.Time `json:"deadline,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NodeDeploy is the Schema for the nodedeploys API
type NodeDeploy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeDeploySpec   `json:"spec,omitempty"`
	Status NodeDeployStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeDeployList contains a list of NodeDeploy
type NodeDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeDeploy{}, &NodeDeployList{})
}
