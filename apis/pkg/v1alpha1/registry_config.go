package v1alpha1

import (
	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +genclient
// +genclient:nonNamespaced

// RegistryConfigMatch is a matcher that determines if a registry should be used.
type RegistryConfigMatch struct {
	// A prefix of the user-specified image name, i.e. using one of the
	// following formats:
	//   - host[:port]
	//  - host[:port]/namespace[/namespace…]
	//  - host[:port]/namespace[/namespace…]/repo
	//  - host[:port]/namespace[/namespace…]/repo(:tag|@digest)
	//  - [*.]host
	// We are following the same format as registries.conf for prefix, see https://www.mankier.com/5/containers-registries.conf
	Prefix string `json:"prefix"`
}

// RegistryConfigCredentials are the credentials to use when accessing a registry.
type RegistryConfigCredentials struct {
	// PullSecretRef is a reference to a secret of type kubernetes.io/dockerconfigjson
	// that contains the credentials to use when accessing this registry.
	PullSecretRef v1.SecretReference `json:"pullSecretRef"`
}

// A ConfigMapKeySelector is a reference to a key of a configmap in an arbitrary
// namespace.
type ConfigMapKeySelector struct {
	// Name of the configmap.
	Name string `json:"name"`
	// Namespace of the configmap.
	Namespace string `json:"namespace"`
	// The key to select.
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

type RegistryConfigSigning struct {
	// Required indicates whether the signature is required for images from this
	// registry.
	Required bool `json:"required"`
	// PublicKeyConfigMapRef is a reference to a ConfigMap that contains the
	// public key to use when verifying signatures from this registry.
	PublicKeyConfigMapRef ConfigMapKeySelector `json:"publicKeyConfigMapRef"`
}

// RegistryConfigSpec defines the desired state of RegistryConfig
type RegistryConfigSpec struct {
	// Match is a list of matchers that determine if this registry should be used.
	Match RegistryConfigMatch `json:"match"`
	// Credentials are the credentials to use when accessing this registry.
	// +optional
	Credentials *RegistryConfigCredentials `json:"credentials,omitempty"`
	// Signing contains the configuration for image signing.
	// +optional
	Signing *RegistryConfigSigning `json:"signing,omitempty"`
}

// A RegistryConfig configures a registry for use by the package manager.
// +kubebuilder:printcolumn:name="PREFIX",type="string",JSONPath=".spec.match.prefix"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster
type RegistryConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegistryConfigSpec `json:"spec,omitempty"`
	// TODO: Add status fields
}

// +kubebuilder:object:root=true

// RegistryConfigList contains a list of RegistryConfig.
type RegistryConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistryConfig `json:"items"`
}
