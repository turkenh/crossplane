package xpkg

import (
	"context"
	sigstorev1alpha1 "github.com/sigstore/policy-controller/pkg/apis/policy/v1alpha1"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane/apis/pkg/v1alpha1"
	webhookcip "github.com/sigstore/policy-controller/pkg/webhook/clusterimagepolicy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegistryConfigStore interface {
	ImagePullSecretFor(ctx context.Context, image string) (string, error)
	ClusterImagePolicyFor(ctx context.Context, image string) (*webhookcip.ClusterImagePolicy, error)
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

func (k *KubeRegistryConfigStore) ClusterImagePolicyFor(ctx context.Context, image string) (*webhookcip.ClusterImagePolicy, error) {
	// List all RegistryConfig objects and find the longest matching prefix
	l := &v1alpha1.RegistryConfigList{}
	if err := k.client.List(ctx, l); err != nil {
		return nil, errors.Wrap(err, "cannot list RegistryConfig objects")
	}

	var rv *v1alpha1.RegistryConfigVerification
	var longestMatch int
	for _, r := range l.Items {
		if strings.HasPrefix(image, r.Spec.Match.Prefix) {
			if r.Spec.Verification != nil && len(r.Spec.Match.Prefix) > longestMatch {
				rv = r.Spec.Verification
				longestMatch = len(r.Spec.Match.Prefix)
			}
		}
	}

	if rv == nil {
		return nil, nil
	}

	return webhookcip.ConvertClusterImagePolicyV1alpha1ToWebhook(&sigstorev1alpha1.ClusterImagePolicy{
		Spec: sigstorev1alpha1.ClusterImagePolicySpec{
			Authorities: rv.Cosign.Authorities,
		},
	}), nil
}
