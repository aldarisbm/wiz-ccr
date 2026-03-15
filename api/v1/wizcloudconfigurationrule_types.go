/*
Copyright 2026.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SeverityType string

const (
	Critical SeverityType = "Critical"
	High     SeverityType = "High"
	Medium   SeverityType = "Medium"
	Low      SeverityType = "Low"
	Info     SeverityType = "Info"
)

type OperationType string

const (
	Create  OperationType = "Create"
	Update  OperationType = "Update"
	Delete  OperationType = "Delete"
	Connect OperationType = "Connect"
)

// MatcherType identifies the kind of matcher used to evaluate the rule.
type MatcherType string

const (
	// MatcherTypeAdmissionsController evaluates resources at Kubernetes admission time.
	MatcherTypeAdmissionsController MatcherType = "ADMISSIONS_CONTROLLER"
)

// WizCloudConfigurationRuleSpec defines the desired state of WizCloudConfigurationRule
type WizCloudConfigurationRuleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	RuleName    *string `json:"rule-name"`
	Description *string `json:"description,omitempty"`
	// +kubebuilder:validation:Enum=Critical;High;Medium;Low;Info
	FindingSeverity     SeverityType      `json:"finding_severity,omitempty"`
	ProjectScope        *string           `json:"project_scope,omitempty"`
	FrameworkCategories []string          `json:"framework_categories,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	TargetNativeType    *string           `json:"target_native_type"`
	// +kubebuilder:validation:Enum=Create;Update;Delete;Connect
	OperationTypes []OperationType `json:"operation_types"`
	// +kubebuilder:validation:Enum=ADMISSIONS_CONTROLLER
	Matchers         []MatcherType `json:"matchers"`
	Code             *string       `json:"code"`
	RemediationSteps *string       `json:"remediation_steps,omitempty"`
}

// WizCloudConfigurationRuleStatus defines the observed state of WizCloudConfigurationRule.
type WizCloudConfigurationRuleStatus struct {
	// wizRuleID is the ID assigned by Wiz when the rule was created.
	// It is used by the controller to update or delete the rule on subsequent reconciles.
	// +optional
	WizRuleID string `json:"wizRuleID,omitempty"`

	// conditions represent the current state of the WizCloudConfigurationRule resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WizCloudConfigurationRule is the Schema for the wizcloudconfigurationrules API
type WizCloudConfigurationRule struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of WizCloudConfigurationRule
	// +required
	Spec WizCloudConfigurationRuleSpec `json:"spec"`

	// status defines the observed state of WizCloudConfigurationRule
	// +optional
	Status WizCloudConfigurationRuleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// WizCloudConfigurationRuleList contains a list of WizCloudConfigurationRule
type WizCloudConfigurationRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []WizCloudConfigurationRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WizCloudConfigurationRule{}, &WizCloudConfigurationRuleList{})
}
