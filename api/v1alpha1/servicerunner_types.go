/*
Copyright 2022 Red Hat.

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

// ServiceRunnerImage defines the image used to manage the underlying service
type ServiceRunnerImage struct {
	CrudImage string `json:"crudImage"`
}

// ServiceRunnerSpec defines the desired state of ServiceRunner
type ServiceRunnerSpec struct {
	// ControlPlaneSecret specifies configuration data for interacting with the control plane
	ControlPlaneSecret string `json:"controlPlaneSecret,omitempty"`

	// ServiceParam contains parameters for the underlying service runner
	ServiceParam map[string]string `json:"serviceParams,omitempty"`

	// ServiceImage specifies the image to use for CRUD operations
	ServiceImage ServiceRunnerImage `json:"serviceImage"`
}

// ServiceRunnerBindingRef contains the secret pointing to binding information
// for workloads.
type ServiceRunnerBindingRef struct {
	// Name contains the name of the secret with binding information.
	Name string `json:"name,omitempty"`
}

// ServiceRunnerStatus defines the observed state of ServiceRunner
type ServiceRunnerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Binding specifies where binding information has been written.
	Binding *ServiceRunnerBindingRef `json:"binding,omitempty"`

	// ObservedGeneration keeps track of the last generation seen by the
	// underlying controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ServiceId sets the ID of the underlying service
	ServiceId string `json:"serviceId,omitempty"`

	// State stores the current state of the runner
	State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceRunner is the Schema for the servicerunners API
type ServiceRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceRunnerSpec   `json:"spec,omitempty"`
	Status ServiceRunnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceRunnerList contains a list of ServiceRunner
type ServiceRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceRunner{}, &ServiceRunnerList{})
}
