/*

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

type SecretRef struct {
	Name *string `json:"name"`
}

type DataFrom struct {
	SecretRef *SecretRef `json:"secretRef,omitempty"`
}

type SecretKeyRef struct {
	Name *string `json:"name"`
	Key  *string `json:"key"`
}

type ValueFrom struct {
	// SecretRef
	// +optional
	SecretRef *SecretRef `json:"secretRef,omitempty"`

	// SecretKeyRef
	// +optional
	SecretKeyRef *SecretKeyRef `json:"secretKeyRef,omitempty"`

	// Template
	// +optional
	Template *string `json:"template,omitempty"`
}

type SecretField struct {
	Name *string `json:"name"`

	// Value
	// +optional
	Value *string `json:"value,omitempty"`

	// ValueFrom
	// +optional
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

type SecretMetadata struct {
	Name              string            `json:"name,omitempty"`
	Namespace         string            `json:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	CreationTimestamp metav1.Time       `json:"creationTimestamp,omitempty"`
}

// SyncedSecretSpec defines the desired state of SyncedSecret
type SyncedSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Secret Metadata
	SecretMetadata SecretMetadata `json:"secretMetadata,omitempty"`
	// IAMRole
	// +optional
	IAMRole *string `json:"IAMRole"`

	// Data
	// +optional
	Data []*SecretField `json:"data,omitempty"`

	// DataFrom
	// +optional
	DataFrom *DataFrom `json:"dataFrom,omitempty"`

	// AWSAccountID
	// +optional
	AWSAccountID *string `json:"AWSAccountID,omitempty"`
}

// SyncedSecretStatus defines the observed state of SyncedSecret
type SyncedSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// this is the version of the secret that is present in k8s secret this should be coming from the local cache
	CurrentVersionID string `json:"currentVersionID"`

	// hash(secret.data) that was generated, used for checking of a Secret has diverged and if it needs reconciling
	SecretHash string `json:"generatedSecretHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SyncedSecret is the Schema for the SyncedSecrets API
type SyncedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyncedSecretSpec   `json:"spec,omitempty"`
	Status SyncedSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SyncedSecretList contains a list of SyncedSecret
type SyncedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyncedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SyncedSecret{}, &SyncedSecretList{})
}
