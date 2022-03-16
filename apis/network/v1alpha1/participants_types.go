/*
Copyright © 2021 Ci4Rail GmbH <engineering@ci4rail.com>
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

// ParticipantsSpec defines the desired state of Participants
type ParticipantsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Component is the name of the component that wants to use the participant.
	Component string `json:"component"`

	// Network is the name of the network the participant is connected to.
	Network string `json:"network"`

	// App is the name of the distrubuted application the participant is created for.
	App string `json:"app"`
}

// ParticipantsStatus defines the observed state of Participants
type ParticipantsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient

// Participants is the Schema for the participants API
type Participants struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ParticipantsSpec   `json:"spec,omitempty"`
	Status ParticipantsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ParticipantsList contains a list of Participants
type ParticipantsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Participants `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Participants{}, &ParticipantsList{})
}
