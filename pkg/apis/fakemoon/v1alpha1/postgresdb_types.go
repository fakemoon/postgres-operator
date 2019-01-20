package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PostgresDBSpec defines the desired state of PostgresDB
type PostgresDBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	PostgresVersion string `json:"version"`
	PostgresPassword string `json:"postgres_password"`
	Storage string `json:"storage_size"`
	StorageClass string `json:"storage_class"`
}

// PostgresDBStatus defines the observed state of PostgresDB
type PostgresDBStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresDB is the Schema for the postgresdbs API
// +k8s:openapi-gen=true
type PostgresDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresDBSpec   `json:"spec,omitempty"`
	Status PostgresDBStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresDBList contains a list of PostgresDB
type PostgresDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresDB{}, &PostgresDBList{})
}
