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

// ArmamentSpec defines the desired state of Armament. Armaments are catalog
// items synced from an external source by the armament-sync controller and
// exposed to consumer workspaces as read-only cached resources.
type ArmamentSpec struct {
	// ExternalID identifies the armament in the external source system and
	// is used by the sync controller to detect updates and deletions.
	ExternalID string `json:"externalID"`

	// DisplayName is a human-readable name shown to consumers.
	DisplayName string `json:"displayName"`

	// Kind classifies the armament (e.g. "revolver", "rifle", "lasso").
	Kind string `json:"kind"`

	// Damage is the armament's damage rating.
	// +optional
	Damage int32 `json:"damage,omitempty"`

	// Range is the armament's effective range in meters.
	// +optional
	Range int32 `json:"range,omitempty"`
}

// ArmamentStatus defines the observed state of Armament.
type ArmamentStatus struct {
	// LastSyncedAt is the time the armament was last reconciled against the
	// external source.
	// +optional
	LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Damage",type=integer,JSONPath=`.spec.damage`
// +kubebuilder:printcolumn:name="Range",type=integer,JSONPath=`.spec.range`

// Armament is the Schema for the armaments catalog.
type Armament struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArmamentSpec   `json:"spec,omitempty"`
	Status ArmamentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArmamentList contains a list of Armament.
type ArmamentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Armament `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Armament{}, &ArmamentList{})
}
