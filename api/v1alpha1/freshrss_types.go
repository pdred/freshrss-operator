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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FreshRSSSpec defines the desired state of FreshRSS
type FreshRSSSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Title is the title for the site.
	Title       string `json:"title,omitempty"`
	DefaultUser string `json:"defaultUser"`
}

// FreshRSSStatus defines the observed state of FreshRSS
type FreshRSSStatus struct {
	URL string `json:"url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FreshRSS is the Schema for the freshrsses API
type FreshRSS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FreshRSSSpec   `json:"spec,omitempty"`
	Status FreshRSSStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FreshRSSList contains a list of FreshRSS
type FreshRSSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FreshRSS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FreshRSS{}, &FreshRSSList{})
}
