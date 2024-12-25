package manager

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	v1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/crossplane/crossplane/apis/pkg/v1beta1"
	"github.com/crossplane/crossplane/pkg/xpkg"
	"github.com/crossplane/crossplane/pkg/xpkg/controller"
	"github.com/crossplane/crossplane/pkg/xpkg/controller/packages"
)

const (
	errCreateK8sClient = "failed to initialize clientset"
	errBuildFetcher    = "cannot build fetcher"
)

// SetupProvider adds a controller that reconciles Providers.
func SetupProvider(mgr ctrl.Manager, o controller.Options) error {
	name := "packages/" + strings.ToLower(v1.ProviderGroupKind)
	np := func() v1.Package { return &v1.Provider{} }
	nr := func() v1.PackageRevision { return &v1.ProviderRevision{} }
	nrl := func() v1.PackageRevisionList { return &v1.ProviderRevisionList{} }

	cs, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, errCreateK8sClient)
	}
	f, err := xpkg.NewK8sFetcher(cs, append(o.FetcherOptions, xpkg.WithNamespace(o.Namespace), xpkg.WithServiceAccount(o.ServiceAccount))...)
	if err != nil {
		return errors.Wrap(err, errBuildFetcher)
	}

	log := o.Logger.WithValues("controller", name)
	opts := []packages.ReconcilerOption{
		packages.WithNewPackageFn(np),
		packages.WithNewPackageRevisionFn(nr),
		packages.WithNewPackageRevisionListFn(nrl),
		packages.WithRevisioner(xpkg.NewPackageRevisioner(f, xpkg.RevisionerWithDefaultRegistry(o.DefaultRegistry))),
		packages.WithConfigStore(xpkg.NewImageConfigStore(mgr.GetClient(), o.Namespace)),
		packages.WithLogger(log),
		packages.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Provider{}).
		Owns(&v1.ProviderRevision{}).
		Watches(&v1beta1.ImageConfig{}, enqueueProvidersForImageConfig(mgr.GetClient(), log)).
		WithOptions(o.ForControllerRuntime()).
		Complete(ratelimiter.NewReconciler(name, errors.WithSilentRequeueOnConflict(packages.NewReconciler(mgr, opts...)), o.GlobalRateLimiter))
}

func enqueueProvidersForImageConfig(kube client.Client, log logging.Logger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		ic, ok := o.(*v1beta1.ImageConfig)
		if !ok {
			return nil
		}
		// We only care about ImageConfigs that have a pull secret.
		if ic.Spec.Registry == nil || ic.Spec.Registry.Authentication == nil || ic.Spec.Registry.Authentication.PullSecretRef.Name == "" {
			return nil
		}
		// Enqueue all Providers matching the prefixes in the ImageConfig.
		l := &v1.ProviderList{}
		if err := kube.List(ctx, l); err != nil {
			// Nothing we can do, except logging, if we can't list Providers.
			log.Debug("Cannot list providers while attempting to enqueue from ImageConfig", "error", err)
			return nil
		}

		var matches []reconcile.Request
		for _, p := range l.Items {
			for _, m := range ic.Spec.MatchImages {
				if strings.HasPrefix(p.GetSource(), m.Prefix) {
					log.Debug("Enqueuing provider for image config", "provider", p.Name, "imageConfig", ic.Name)
					matches = append(matches, reconcile.Request{NamespacedName: types.NamespacedName{Name: p.Name}})
				}
			}
		}
		return matches
	})
}
