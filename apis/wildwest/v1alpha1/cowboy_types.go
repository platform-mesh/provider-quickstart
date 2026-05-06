/*
Copyright 2025 The Platform Mesh Authors.

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

// CowboySpec defines the desired state of Cowboy
type CowboySpec struct {
	// Intent is the desired action for the cowboy
	// +optional
	Intent string `json:"intent,omitempty"`

	// SecretRefs is an array of references to Secrets containing cowboy credentials
	// +optional
	SecretRefs []SecretReference `json:"secretRefs,omitempty"`
}

// SecretReference references a Secret by name in the same namespace as the Cowboy.
type SecretReference struct {
	// Name of the referenced Secret
	Name string `json:"name"`
}

// CowboyStatus defines the observed state of Cowboy
type CowboyStatus struct {
	// Result is the outcome of the cowboy's action
	// +optional
	Result string `json:"result,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Intent",type=string,JSONPath=`.spec.intent`
// +kubebuilder:printcolumn:name="Result",type=string,JSONPath=`.status.result`

// Cowboy is the Schema for the cowboys API
type Cowboy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CowboySpec   `json:"spec,omitempty"`
	Status CowboyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CowboyList contains a list of Cowboy
type CowboyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cowboy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cowboy{}, &CowboyList{})
}
