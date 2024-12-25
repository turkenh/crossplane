/*
Copyright 2020 The Crossplane Authors.

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

// Package manager contains the controller setup functions for the package
// manager package controllers.
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

// SetupConfiguration adds a controller that reconciles Configurations.
func SetupConfiguration(mgr ctrl.Manager, o controller.Options) error {
	name := "packages/" + strings.ToLower(v1.ConfigurationGroupKind)
	np := func() v1.Package { return &v1.Configuration{} }
	nr := func() v1.PackageRevision { return &v1.ConfigurationRevision{} }
	nrl := func() v1.PackageRevisionList { return &v1.ConfigurationRevisionList{} }

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to initialize clientset")
	}
	fetcher, err := xpkg.NewK8sFetcher(clientset, append(o.FetcherOptions, xpkg.WithNamespace(o.Namespace), xpkg.WithServiceAccount(o.ServiceAccount))...)
	if err != nil {
		return errors.Wrap(err, "cannot build fetcher")
	}

	log := o.Logger.WithValues("controller", name)
	r := packages.NewReconciler(mgr,
		packages.WithNewPackageFn(np),
		packages.WithNewPackageRevisionFn(nr),
		packages.WithNewPackageRevisionListFn(nrl),
		packages.WithRevisioner(xpkg.NewPackageRevisioner(fetcher, xpkg.RevisionerWithDefaultRegistry(o.DefaultRegistry))),
		packages.WithConfigStore(xpkg.NewImageConfigStore(mgr.GetClient(), o.Namespace)),
		packages.WithLogger(log),
		packages.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Configuration{}).
		Owns(&v1.ConfigurationRevision{}).
		Watches(&v1beta1.ImageConfig{}, enqueueConfigurationsForImageConfig(mgr.GetClient(), log)).
		WithOptions(o.ForControllerRuntime()).
		Complete(ratelimiter.NewReconciler(name, errors.WithSilentRequeueOnConflict(r), o.GlobalRateLimiter))
}

func enqueueConfigurationsForImageConfig(kube client.Client, log logging.Logger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		ic, ok := o.(*v1beta1.ImageConfig)
		if !ok {
			return nil
		}
		// We only care about ImageConfigs that have a pull secret.
		if ic.Spec.Registry == nil || ic.Spec.Registry.Authentication == nil || ic.Spec.Registry.Authentication.PullSecretRef.Name == "" {
			return nil
		}
		// Enqueue all Configurations matching the prefixes in the ImageConfig.
		l := &v1.ConfigurationList{}
		if err := kube.List(ctx, l); err != nil {
			// Nothing we can do, except logging, if we can't list Configurations.
			log.Debug("Cannot list configurations while attempting to enqueue from ImageConfig", "error", err)
			return nil
		}

		var matches []reconcile.Request
		for _, c := range l.Items {
			for _, m := range ic.Spec.MatchImages {
				if strings.HasPrefix(c.GetSource(), m.Prefix) {
					log.Debug("Enqueuing configuration for image config", "configuration", c.Name, "imageConfig", ic.Name)
					matches = append(matches, reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name}})
				}
			}
		}
		return matches
	})
}
