/*
Copyright 2021 Crunchy Data Solutions, Inc.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BridgeServiceSpec defines the desired state of BridgeService
type BridgeServiceSpec struct {

	// The secret storing the crunchy bridge api connection credentials(application id and secret) to use for generating token with the API endpoint. The secret may be placed in a separate NamespacedName
	CredentialsRef string `json:"credentialsRef"`
}

// BridgeServiceStatus defines the observed state of BridgeService
type BridgeServiceStatus struct {

	// A list of instances returned from querying the API
	Instances []Instance `json:"instances,omitempty"`
}

type Instance struct {
	// The ID of the Cluster
	ID string `json:"id,omitempty"`

	// The Name of the Cluster
	Name string `json:"name,omitempty"`

	// other  information related to this instance
	InstanceInfo InstanceInfo `json:"extraInfo,omitempty"`
}

type InstanceInfo struct {

	// ProviderId is the cloud provider Id
	ProviderId string `json:"provider_id,omitempty"`

	// RegionID is the region of cloud provider
	RegionId string `json:"region_id,omitempty"`

	// the type of the instance (‘primary’ or ‘read_replica’)
	Type string `json:"type,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BridgeService is the Schema for the bridgeservices API
type BridgeService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeServiceSpec   `json:"spec,omitempty"`
	Status BridgeServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BridgeServiceList contains a list of BridgeService
type BridgeServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BridgeService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BridgeService{}, &BridgeServiceList{})
}
