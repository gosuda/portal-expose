/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TunnelClassSpec defines the desired state of TunnelClass
type TunnelClassSpec struct {
	// Replicas is the number of tunnel pod replicas
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// Size defines the resource allocation tier: small | medium | large
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=small;medium;large
	Size string `json:"size"`

	// NodeSelector constrains tunnel pods to nodes with specific labels
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations allows tunnel pods to schedule on tainted nodes
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// TunnelClassStatus defines the observed state of TunnelClass.
type TunnelClassStatus struct {
	// ObservedGeneration is the last observed spec generation
	// Used for change detection
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the current state of the TunnelClass resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TunnelClass is the Schema for the tunnelclasses API
type TunnelClass struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TunnelClass
	// +required
	Spec TunnelClassSpec `json:"spec"`

	// status defines the observed state of TunnelClass
	// +optional
	Status TunnelClassStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TunnelClassList contains a list of TunnelClass
type TunnelClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TunnelClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TunnelClass{}, &TunnelClassList{})
}
