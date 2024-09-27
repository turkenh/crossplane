package xpkg

import (
	"context"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane/apis/pkg/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type RegistryConfigStore interface {
	ImagePullSecretFor(ctx context.Context, image string) (string, error)
	SigningPublicKeyFor(ctx context.Context, image string) (string, error)
}

type KubernetesRegistryConfigStoreOption func(*KubeRegistryConfigStore)

func KubernetesRegistryConfigStoreWithLogger(l logging.Logger) KubernetesRegistryConfigStoreOption {
	return func(k *KubeRegistryConfigStore) {
		k.log = l
	}
}

func NewKubeRegistryConfigStore(client client.Client, namespace string, opts ...KubernetesRegistryConfigStoreOption) *KubeRegistryConfigStore {
	k := KubeRegistryConfigStore{client: client, namespace: namespace}

	for _, o := range opts {
		o(&k)
	}

	return &k
}

type KubeRegistryConfigStore struct {
	client    client.Client
	namespace string

	log logging.Logger
}

func (k *KubeRegistryConfigStore) ImagePullSecretFor(ctx context.Context, image string) (string, error) {
	// TODO: Do we need a cache here? Or controller runtime cache is enough?
	// TODO: Refactor the logic fetching the longest matching prefix to a helper
	//  function and share it between the two methods. But longest matching
	//  prefix might change based on the use case, e.g. no signing key in one
	//  match but in another.
	// List all RegistryConfig objects and find the longest matching prefix
	l := &v1alpha1.RegistryConfigList{}
	if err := k.client.List(ctx, l); err != nil {
		return "", errors.Wrap(err, "cannot list RegistryConfig objects")
	}

	var secret string
	var longestMatch int
	for _, r := range l.Items {
		if strings.HasPrefix(image, r.Spec.Match.Prefix) {
			if r.Spec.Credentials != nil && len(r.Spec.Match.Prefix) > longestMatch {
				secret = r.Spec.Credentials.PullSecretRef.Name
				longestMatch = len(r.Spec.Match.Prefix)
			}
		}
	}

	return secret, nil
}

func (k *KubeRegistryConfigStore) SigningPublicKeyFor(ctx context.Context, image string) (string, error) {
	// List all RegistryConfig objects and find the longest matching prefix
	l := &v1alpha1.RegistryConfigList{}
	if err := k.client.List(ctx, l); err != nil {
		return "", errors.Wrap(err, "cannot list RegistryConfig objects")
	}

	var cmName, cmKey string
	var longestMatch int
	for _, r := range l.Items {
		if strings.HasPrefix(image, r.Spec.Match.Prefix) {
			if r.Spec.Signing != nil && len(r.Spec.Match.Prefix) > longestMatch {
				cmName = r.Spec.Signing.PublicKeyConfigMapRef.Name
				cmKey = r.Spec.Signing.PublicKeyConfigMapRef.Key
				longestMatch = len(r.Spec.Match.Prefix)
			}
		}
	}

	if cmName == "" {
		return "", nil
	}

	// Fetch the public key from the ConfigMap
	cm := &corev1.ConfigMap{}
	if err := k.client.Get(ctx, client.ObjectKey{Namespace: k.namespace, Name: cmName}, cm); err != nil {
		return "", errors.Wrapf(err, "cannot get ConfigMap %q", cmName)
	}
	if cm.Data[cmKey] == "" {
		return "", errors.Errorf("public key %q not found in ConfigMap", cmKey)
	}

	return cm.Data[cmKey], nil
}
