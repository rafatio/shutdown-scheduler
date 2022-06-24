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
	// +kubebuilder:validation:Enum:0;1;2;3;4;5;6
	WeekDay int    `json:"weekDay,omitempty"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

// ShutdownSchedulerSpec defines the desired state of ShutdownScheduler
type ShutdownSchedulerSpec struct {
	Resource  *corev1.ObjectReference `json:"resource,omitempty"`
	TimeRange []*TimeRange            `json:"timeRange,omitempty"`
}

// ShutdownSchedulerStatus defines the observed state of ShutdownScheduler
type ShutdownSchedulerStatus struct {
	PreviousReplicas int `json:"previousReplicas,omitempty"`
	// +optional
	Shutdown bool `json:"shutdown"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ShutdownScheduler is the Schema for the shutdownschedulers API
type ShutdownScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShutdownSchedulerSpec   `json:"spec,omitempty"`
	Status ShutdownSchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ShutdownSchedulerList contains a list of ShutdownScheduler
type ShutdownSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShutdownScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShutdownScheduler{}, &ShutdownSchedulerList{})
}
