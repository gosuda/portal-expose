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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppSpec defines the application to expose
type AppSpec struct {
	// Name is the application name (becomes subdomain)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Service references the Kubernetes Service to expose
	// +kubebuilder:validation:Required
	Service ServiceRef `json:"service"`
}

// ServiceRef references a Kubernetes Service
type ServiceRef struct {
	// Name is the Service name in the same namespace
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Port is the Service port number to expose
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
}

// RelayTarget defines a Portal relay endpoint
type RelayTarget struct {
	// Name is the relay identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// URL is the WebSocket relay URL
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^wss://.*`
	URL string `json:"url"`
}

// RelaySpec defines relay configuration
type RelaySpec struct {
	// Targets is the list of Portal relay endpoints
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Targets []RelayTarget `json:"targets"`
}

// PortalExposeSpec defines the desired state of PortalExpose
type PortalExposeSpec struct {
	// App defines which application to expose
	// +kubebuilder:validation:Required
	App AppSpec `json:"app"`

	// Relay defines relay configuration
	// +kubebuilder:validation:Required
	Relay RelaySpec `json:"relay"`

	// TunnelClassName references the TunnelClass to use
	// Uses default TunnelClass if omitted
	// +optional
	TunnelClassName string `json:"tunnelClassName,omitempty"`
}

// TunnelPodStatus represents tunnel pod readiness
type TunnelPodStatus struct {
	// Ready is the number of ready tunnel pods
	// +optional
	Ready int32 `json:"ready,omitempty"`

	// Total is the desired number of tunnel pods
	// +optional
	Total int32 `json:"total,omitempty"`
}

// RelayConnectionStatus represents relay connection state
type RelayConnectionStatus struct {
	// Name is the relay name (matches spec.relay.targets[].name)
	// +required
	Name string `json:"name"`

	// Status is the connection status: Connected | Disconnected | Unknown
	// +kubebuilder:validation:Enum=Connected;Disconnected;Unknown
	// +required
	Status string `json:"status"`

	// ConnectedAt is when the connection was established
	// +optional
	ConnectedAt *metav1.Time `json:"connectedAt,omitempty"`

	// LastError is the last connection error message
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// RelayStatus represents the state of all relay connections
type RelayStatus struct {
	// Connected lists per-relay connection states
	// +optional
	Connected []RelayConnectionStatus `json:"connected,omitempty"`
}

// PortalExposeStatus defines the observed state of PortalExpose.
type PortalExposeStatus struct {
	// Phase is the current state: Pending | Ready | Degraded | Failed
	// +kubebuilder:validation:Enum=Pending;Ready;Degraded;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// PublicURL is the accessible endpoint (e.g., "https://my-app.portal.gosuda.org")
	// +optional
	PublicURL string `json:"publicURL,omitempty"`

	// TunnelPods shows tunnel pod readiness
	// +optional
	TunnelPods TunnelPodStatus `json:"tunnelPods,omitempty"`

	// Relay shows relay connection status
	// +optional
	Relay RelayStatus `json:"relay,omitempty"`

	// Conditions represent the current state of the PortalExpose resource.
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "TunnelDeploymentReady": all tunnel pods are ready
	// - "RelayConnected": all relays are connected
	// - "ServiceExists": referenced Service was found
	//
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PortalExpose is the Schema for the portalexposes API
type PortalExpose struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PortalExpose
	// +required
	Spec PortalExposeSpec `json:"spec"`

	// status defines the observed state of PortalExpose
	// +optional
	Status PortalExposeStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PortalExposeList contains a list of PortalExpose
type PortalExposeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PortalExpose `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PortalExpose{}, &PortalExposeList{})
}
