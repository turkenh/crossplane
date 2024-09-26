package config

import (
	"context"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane/apis/pkg/v1alpha1"

	"github.com/crossplane/crossplane/internal/controller/pkg/controller"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// Reconciler reconciles packages.
type Reconciler struct {
	client client.Client
	log    logging.Logger
}

// Setup adds a controller that reconciles the Lock.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := "packages/" + strings.ToLower(v1alpha1.RegistryConfigGroupKind)

	r := NewReconciler(mgr,
		WithLogger(o.Logger.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.RegistryConfig{}).
		WithOptions(o.ForControllerRuntime()).
		Complete(ratelimiter.NewReconciler(name, errors.WithSilentRequeueOnConflict(r), o.GlobalRateLimiter))
}

// NewReconciler creates a new registry config reconciler.
func NewReconciler(mgr manager.Manager, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client: mgr.GetClient(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Reconcile a registry config.
func (r *Reconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("Reconciling registry config", "request", req)
	return ctrl.Result{}, nil
}
