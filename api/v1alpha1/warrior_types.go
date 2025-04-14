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

// WarriorSpec defines the desired state of Warrior
type WarriorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Project    string `json:"project"`
	Downloader string `json:"downloader,omitempty"`
	// +kubebuilder:default:={"maximum":1,"concurrency":5}
	Scaling WarriorScaling `json:"scaling,omitempty"`
	// +kubebuilder:default:={"limits":{"cpu":"0","memory":"0"},"requests":{"cpu":"0","memory":"0"}}
	Resources WarriorResources `json:"resources,omitempty"`
}

type WarriorScaling struct {
	// +kubebuilder:default:=0
	Minimum int `json:"minimum,omitempty"`
	// +kubebuilder:default:=0
	Maximum int `json:"maximum"`
	// +kubebuilder:default:=5
	Concurrency int `json:"concurrency"`
}

type WarriorResources struct {
	Limits   Resources `json:"limits"`
	Requests Resources `json:"requests"`
}

type Resources struct {
	// +kubebuilder:default:="0"
	CPU string `json:"cpu"`
	// +kubebuilder:default:="0"
	Memory string `json:"memory"`
}

// WarriorStatus defines the observed state of Warrior
type WarriorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	PodCount   int                `json:"pod_count"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Warrior is the Schema for the warriors API
// //+kubebuilder:subresource:status
type Warrior struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WarriorSpec   `json:"spec,omitempty"`
	Status WarriorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WarriorList contains a list of Warrior
type WarriorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Warrior `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Warrior{}, &WarriorList{})
}
