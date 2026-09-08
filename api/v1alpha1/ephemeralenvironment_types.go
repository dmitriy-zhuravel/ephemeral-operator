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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EphemeralEnvironmentSpec defines the desired state of EphemeralEnvironment
// +kubebuilder:validation:Required
type EphemeralEnvironmentSpec struct {

	// +kubebuilder:validation:Required
	TTL *metav1.Duration `json:"ttl"`

	// +kubebuilder:validation:Required
	TargetNamespace *string `json:"targetNamespace,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ScaleToZero;Delete
	Action string `json:"action,omitempty"`
}

// EphemeralEnvironmentStatus defines the observed state of EphemeralEnvironment.
type EphemeralEnvironmentStatus struct {
	State      string       `json:"state,omitempty"`
	ExpiryTime *metav1.Time `json:"expiryTime,omitempty"`
	Reason     string       `json:"reason,omitempty"`
}

// EphemeralEnvironment is the Schema for the ephemeralenvironments API
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="ExpiryTime",type=date,JSONPath=`.status.expiryTime`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
// +kubebuilder:printcolumn:name="TargetNS",type=string,JSONPath=`.spec.targetNamespace`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

type EphemeralEnvironment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of EphemeralEnvironment
	// +required
	Spec EphemeralEnvironmentSpec `json:"spec"`

	// status defines the observed state of EphemeralEnvironment
	// +optional
	Status EphemeralEnvironmentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// EphemeralEnvironmentList contains a list of EphemeralEnvironment
type EphemeralEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []EphemeralEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EphemeralEnvironment{}, &EphemeralEnvironmentList{})
}
