/*
Copyright 2024 The Crossplane Authors.

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

// Package signature implements the controller verifying package signatures.
package signature

import (
	"context"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	v1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/crossplane/crossplane/internal/controller/pkg/controller"
	"github.com/crossplane/crossplane/internal/xpkg"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	cosign "github.com/sigstore/policy-controller/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

const (
	reconcileTimeout = 3 * time.Minute
)

const (
	errGetPackageRevision  = "cannot get package revision"
	errParseReference      = "cannot parse package image reference"
	errNewKubernetesClient = "cannot create new Kubernetes clientset"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// WithNewPackageRevisionFn determines the type of package being reconciled.
func WithNewPackageRevisionFn(f func() v1.PackageRevision) ReconcilerOption {
	return func(r *Reconciler) {
		r.newPackageRevision = f
	}
}

// WithConfigStore specifies the ConfigStore to use for fetching image
// configurations.
func WithConfigStore(c xpkg.ConfigStore) ReconcilerOption {
	return func(r *Reconciler) {
		r.config = c
	}
}

// WithNamespace specifies the namespace in which the Reconciler should create
// runtime resources.
func WithNamespace(n string) ReconcilerOption {
	return func(r *Reconciler) {
		r.namespace = n
	}
}

// WithDefaultRegistry specifies the registry to use for fetching images.
func WithDefaultRegistry(registry string) ReconcilerOption {
	return func(r *Reconciler) {
		r.registry = registry
	}
}

// WithServiceAccount specifies the service account to use for fetching images.
func WithServiceAccount(sa string) ReconcilerOption {
	return func(r *Reconciler) {
		r.serviceAccount = sa
	}
}

// Reconciler reconciles package revision.
type Reconciler struct {
	client         client.Client
	clientset      kubernetes.Interface
	config         xpkg.ConfigStore
	log            logging.Logger
	serviceAccount string
	namespace      string
	registry       string

	newPackageRevision func() v1.PackageRevision
}

// SetupProviderRevision adds a controller that reconciles ProviderRevisions.
func SetupProviderRevision(mgr ctrl.Manager, o controller.Options) error {
	name := "package-signature-verification/" + strings.ToLower(v1.ProviderRevisionGroupKind)
	nr := func() v1.PackageRevision { return &v1.ProviderRevision{} }

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, errNewKubernetesClient)
	}

	log := o.Logger.WithValues("controller", name)
	cb := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		// TODO: Watch for changes in ImageConfig and trigger reconciliation of ProviderRevisions
		For(&v1.ProviderRevision{})

	ro := []ReconcilerOption{
		WithNewPackageRevisionFn(nr),
		WithConfigStore(xpkg.NewImageConfigStore(mgr.GetClient())),
		WithLogger(log),
		WithNamespace(o.Namespace),
		WithDefaultRegistry(o.DefaultRegistry),
		WithServiceAccount(o.ServiceAccount),
	}

	return cb.WithOptions(o.ForControllerRuntime()).
		Complete(ratelimiter.NewReconciler(name, errors.WithSilentRequeueOnConflict(NewReconciler(mgr.GetClient(), clientset, ro...)), o.GlobalRateLimiter))
}

// TODO: Setup ConfigurationRevision controller
// TODO: Setup FunctionRevision controller

// NewReconciler creates a new package revision reconciler.
func NewReconciler(client client.Client, clientset kubernetes.Interface, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client:    client,
		clientset: clientset,
		log:       logging.NewNopLogger(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile package revision.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { //nolint:gocognit // Reconcilers are often very complex.
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	pr := r.newPackageRevision()
	if err := r.client.Get(ctx, req.NamespacedName, pr); err != nil {
		log.Debug(errGetPackageRevision, "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetPackageRevision)
	}

	log = log.WithValues(
		"uid", pr.GetUID(),
		"version", pr.GetResourceVersion(),
		"name", pr.GetName(),
	)

	// If signature verification is already complete nothing to do.
	if cond := pr.GetCondition(v1.TypeSignatureVerificationComplete); cond.Status == corev1.ConditionTrue {
		return reconcile.Result{}, nil
	}

	ic, vc, err := r.config.ImageVerificationConfigFor(ctx, pr.GetSource())
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "cannot get image verification config")
	}
	if vc == nil || vc.CosignConfig == nil {
		// No verification config found for this image, so, we will skip
		// verification.
		pr.SetConditions(v1.VerificationSkipped())
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, pr), "cannot update package revision status")
	}

	ref, err := name.ParseReference(pr.GetSource(), name.WithDefaultRegistry(r.registry))
	if err != nil {
		pr.SetConditions(v1.VerificationInComplete(errors.Wrap(err, errParseReference)))
		_ = r.client.Status().Update(ctx, pr)
		return reconcile.Result{}, errors.Wrap(err, errParseReference)
	}

	var pullSecrets []string
	for _, s := range pr.GetPackagePullSecrets() {
		pullSecrets = append(pullSecrets, s.Name)
	}

	_, s, err := r.config.PullSecretFor(ctx, pr.GetSource())
	if err != nil {
		pr.SetConditions(v1.VerificationInComplete(errors.Wrap(err, "cannot get image config pull secret for image")))
		_ = r.client.Status().Update(ctx, pr)
		return reconcile.Result{}, errors.Wrap(err, "cannot get image config pull secret for image")
	}
	if s != "" {
		pullSecrets = append(pullSecrets, s)
	}

	auth, err := k8schain.New(ctx, r.clientset, k8schain.Options{
		Namespace:          r.namespace,
		ServiceAccountName: r.serviceAccount,
		ImagePullSecrets:   pullSecrets,
	})

	res, errs := cosign.ValidatePolicy(ctx, r.namespace, ref, *vc.CosignConfig, auth)
	if res != nil {
		// Ignore the errors for other authorities if we got a policy result.
		pr.SetConditions(v1.VerificationSucceeded(ic))
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, pr), "cannot update status with successful verification")
	}

	// TODO: Polish error rendering

	pr.SetConditions(v1.VerificationFailed(ic, errs))
	return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, pr), "cannot update status with failed verification")
}
