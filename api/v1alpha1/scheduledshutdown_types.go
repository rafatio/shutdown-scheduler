/*
Copyright 2022.

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

type TimeRange struct {
	Start metav1.Time `json:"start,omitempty"`
	End   metav1.Time `json:"end,omitempty"`
}

// ScheduledShutdownSpec defines the desired state of ScheduledShutdown
type ScheduledShutdownSpec struct {
	Resource  *corev1.ObjectReference `json:"resource,omitempty"`
	TimeRange []*TimeRange            `json:"timeRange,omitempty"`
}

// ScheduledShutdownStatus defines the observed state of ScheduledShutdown
type ScheduledShutdownStatus struct {
	PreviousReplicas int  `json:"previousReplicas,omitempty"`
	Shutdown         bool `json:"shutdown,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ScheduledShutdown is the Schema for the scheduledshutdowns API
type ScheduledShutdown struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledShutdownSpec   `json:"spec,omitempty"`
	Status ScheduledShutdownStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ScheduledShutdownList contains a list of ScheduledShutdown
type ScheduledShutdownList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScheduledShutdown `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScheduledShutdown{}, &ScheduledShutdownList{})
}
