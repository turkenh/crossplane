// Package packages contains the reconciler for the Package resources.
package packages

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	v1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/crossplane/crossplane/pkg/xpkg"
)

const (
	reconcileTimeout = 1 * time.Minute

	// pullWait is the time after which the package manager will check for
	// updated content for the given package reference. This behavior is only
	// enabled when the packagePullPolicy is Always.
	pullWait = 1 * time.Minute

	reconcilePausedMsg = "Reconciliation (including deletion) is paused via the pause annotation"
)

func pullBasedRequeue(p *corev1.PullPolicy) reconcile.Result {
	if p != nil && *p == corev1.PullAlways {
		return reconcile.Result{RequeueAfter: pullWait}
	}
	return reconcile.Result{Requeue: false}
}

const (
	errGetPackage           = "cannot get package"
	errListRevisions        = "cannot list revisions for package"
	errUnpack               = "cannot unpack package"
	errApplyPackageRevision = "cannot apply package revision"
	errGCPackageRevision    = "cannot garbage collect old package revision"
	errGetPullConfig        = "cannot get image pull secret from config"

	errUpdateStatus                  = "cannot update package status"
	errUpdateInactivePackageRevision = "cannot update inactive package revision"

	errUnhealthyPackageRevision     = "current package revision is unhealthy"
	errUnknownPackageRevisionHealth = "current package revision health is unknown"
)

// Event reasons.
const (
	reasonList               event.Reason = "ListRevision"
	reasonUnpack             event.Reason = "UnpackPackage"
	reasonTransitionRevision event.Reason = "TransitionRevision"
	reasonGarbageCollect     event.Reason = "GarbageCollect"
	reasonInstall            event.Reason = "InstallPackageRevision"
	reasonPaused             event.Reason = "ReconciliationPaused"
	reasonImageConfig        event.Reason = "ImageConfigSelection"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithNewPackageFn determines the type of package being reconciled.
func WithNewPackageFn(f func() v1.Package) ReconcilerOption {
	return func(r *Reconciler) {
		r.newPackage = f
	}
}

// WithNewPackageRevisionFn determines the type of package being reconciled.
func WithNewPackageRevisionFn(f func() v1.PackageRevision) ReconcilerOption {
	return func(r *Reconciler) {
		r.newPackageRevision = f
	}
}

// WithNewPackageRevisionListFn determines the type of package being reconciled.
func WithNewPackageRevisionListFn(f func() v1.PackageRevisionList) ReconcilerOption {
	return func(r *Reconciler) {
		r.newPackageRevisionList = f
	}
}

// WithRevisioner specifies how the Reconciler should acquire a package image's
// revision name.
func WithRevisioner(d xpkg.Revisioner) ReconcilerOption {
	return func(r *Reconciler) {
		r.pkg = d
	}
}

// WithConfigStore specifies the image config store to use.
func WithConfigStore(c xpkg.ConfigStore) ReconcilerOption {
	return func(r *Reconciler) {
		r.config = c
	}
}

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// WithRecorder specifies how the Reconciler should record Kubernetes events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

// Reconciler reconciles packages.
type Reconciler struct {
	client resource.ClientApplicator
	pkg    xpkg.Revisioner
	config xpkg.ConfigStore
	log    logging.Logger
	record event.Recorder

	newPackage             func() v1.Package
	newPackageRevision     func() v1.PackageRevision
	newPackageRevisionList func() v1.PackageRevisionList
}

// NewReconciler creates a new package reconciler.
func NewReconciler(mgr ctrl.Manager, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client: resource.ClientApplicator{
			Client:     mgr.GetClient(),
			Applicator: resource.NewAPIPatchingApplicator(mgr.GetClient()),
		},
		pkg:    xpkg.NewNopRevisioner(),
		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile package.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { //nolint:gocognit // Reconcilers are complex. Be wary of adding more.
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	p := r.newPackage()
	if err := r.client.Get(ctx, req.NamespacedName, p); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise
		// we'll be requeued implicitly because we return an error.
		log.Debug(errGetPackage, "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetPackage)
	}

	// Check the pause annotation and return if it has the value "true"
	// after logging, publishing an event and updating the SYNC status condition
	if meta.IsPaused(p) {
		r.record.Event(p, event.Normal(reasonPaused, reconcilePausedMsg))
		p.SetConditions(xpv1.ReconcilePaused().WithMessage(reconcilePausedMsg))
		// If the pause annotation is removed, we will have a chance to reconcile again and resume
		// and if status update fails, we will reconcile again to retry to update the status
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, p), errUpdateStatus)
	}
	if c := p.GetCondition(xpv1.ReconcilePaused().Type); c.Reason == xpv1.ReconcilePaused().Reason {
		p.CleanConditions()
		// Persist the removal of conditions and return. We'll be requeued
		// with the updated status and resume reconciliation.
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, p), errUpdateStatus)
	}

	// Get existing package revisions.
	prs := r.newPackageRevisionList()
	if err := r.client.List(ctx, prs, client.MatchingLabels(map[string]string{v1.LabelParentPackage: p.GetName()})); resource.IgnoreNotFound(err) != nil {
		err = errors.Wrap(err, errListRevisions)
		r.record.Event(p, event.Warning(reasonList, err))
		return reconcile.Result{}, err
	}

	imageConfig, pullSecretFromConfig, err := r.config.PullSecretFor(ctx, p.GetSource())
	if err != nil {
		err = errors.Wrap(err, errGetPullConfig)
		p.SetConditions(v1.Unpacking().WithMessage(err.Error()))
		_ = r.client.Status().Update(ctx, p)

		r.record.Event(p, event.Warning(reasonImageConfig, err))

		return reconcile.Result{}, err
	}

	var secrets []string
	if pullSecretFromConfig != "" {
		secrets = append(secrets, pullSecretFromConfig)
	}
	revisionName, err := r.pkg.Revision(ctx, p, secrets...)
	if err != nil {
		err = errors.Wrap(err, errUnpack)
		p.SetConditions(v1.Unpacking().WithMessage(err.Error()))
		r.record.Event(p, event.Warning(reasonUnpack, err))

		if updateErr := r.client.Status().Update(ctx, p); updateErr != nil {
			return reconcile.Result{}, errors.Wrap(updateErr, errUpdateStatus)
		}

		return reconcile.Result{}, err
	}

	if revisionName == "" {
		p.SetConditions(v1.Unpacking().WithMessage("Waiting for unpack to complete"))
		r.record.Event(p, event.Normal(reasonUnpack, "Waiting for unpack to complete"))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, p), errUpdateStatus)
	}

	// Set the current revision and identifier.
	p.SetCurrentRevision(revisionName)
	p.SetCurrentIdentifier(p.GetSource())

	pr := r.newPackageRevision()
	maxRevision := int64(0)
	oldestRevision := int64(math.MaxInt64)
	oldestRevisionIndex := -1
	revisions := prs.GetRevisions()

	// Check to see if revision already exists.
	for index, rev := range revisions {
		revisionNum := rev.GetRevision()

		// Set max revision to the highest numbered existing revision.
		if revisionNum > maxRevision {
			maxRevision = revisionNum
		}

		// Set oldest revision to the lowest numbered revision and
		// record its index.
		if revisionNum < oldestRevision {
			oldestRevision = revisionNum
			oldestRevisionIndex = index
		}
		// If revision name is same as current revision, then revision
		// already exists.
		if rev.GetName() == p.GetCurrentRevision() {
			pr = rev
			// Finish iterating through all revisions to make sure
			// all non-current revisions are inactive.
			continue
		}
		if rev.GetDesiredState() == v1.PackageRevisionActive {
			// If revision is not the current revision, set to
			// inactive. This should always be done, regardless of
			// the package's revision activation policy.
			rev.SetDesiredState(v1.PackageRevisionInactive)
			if err := r.client.Apply(ctx, rev, resource.MustBeControllableBy(p.GetUID())); err != nil {
				if kerrors.IsConflict(err) {
					return reconcile.Result{Requeue: true}, nil
				}
				err = errors.Wrap(err, errUpdateInactivePackageRevision)
				r.record.Event(p, event.Warning(reasonTransitionRevision, err))
				return reconcile.Result{}, err
			}
		}
	}

	// The current revision should always be the highest numbered revision.
	if pr.GetRevision() < maxRevision || maxRevision == 0 {
		pr.SetRevision(maxRevision + 1)
	}

	// Check to see if there are revisions eligible for garbage collection.
	if p.GetRevisionHistoryLimit() != nil &&
		*p.GetRevisionHistoryLimit() != 0 &&
		len(revisions) > (int(*p.GetRevisionHistoryLimit())+1) {
		gcRev := revisions[oldestRevisionIndex]
		// Find the oldest revision and delete it.
		if err := r.client.Delete(ctx, gcRev); err != nil {
			err = errors.Wrap(err, errGCPackageRevision)
			r.record.Event(p, event.Warning(reasonGarbageCollect, err))
			return reconcile.Result{}, err
		}
	}

	// TODO(phisco): refactor these conditions to make it clearer
	if pr.GetCondition(v1.TypeHealthy).Status == corev1.ConditionTrue {
		if p.GetCondition(v1.TypeHealthy).Status != corev1.ConditionTrue {
			// NOTE(phisco): We don't want to spam the user with events if the
			// package is already healthy.
			r.record.Event(p, event.Normal(reasonInstall, "Successfully installed package revision"))
		}
		p.SetConditions(v1.Healthy())
	}
	if prHealthy := pr.GetCondition(v1.TypeHealthy); prHealthy.Status == corev1.ConditionFalse {
		p.SetConditions(v1.Unhealthy().WithMessage(prHealthy.Message))
		r.record.Event(p, event.Warning(reasonInstall, errors.New(errUnhealthyPackageRevision)))
	}
	if prHealthy := pr.GetCondition(v1.TypeHealthy); prHealthy.Status == corev1.ConditionUnknown {
		p.SetConditions(v1.UnknownHealth().WithMessage(prHealthy.Message))
		r.record.Event(p, event.Warning(reasonInstall, errors.New(errUnknownPackageRevisionHealth)))
	}

	if pr.GetUID() == "" && imageConfig != "" {
		// We only record this event if the revision is new, as we don't want to
		// spam the user with events if the revision already exists.
		log.Debug("Selected pull secret from image config store", "image", p.GetSource(), "imageConfig", imageConfig, "pullSecret", pullSecretFromConfig)
		r.record.Event(p, event.Normal(reasonImageConfig, fmt.Sprintf("Selected pullSecret %q from ImageConfig %q for registry authentication", pullSecretFromConfig, imageConfig)))
	}

	// Create the non-existent package revision.
	pr.SetName(revisionName)
	pr.SetLabels(map[string]string{v1.LabelParentPackage: p.GetName()})
	pr.SetSource(p.GetSource())
	pr.SetPackagePullPolicy(p.GetPackagePullPolicy())
	pr.SetPackagePullSecrets(p.GetPackagePullSecrets())
	pr.SetIgnoreCrossplaneConstraints(p.GetIgnoreCrossplaneConstraints())
	pr.SetSkipDependencyResolution(p.GetSkipDependencyResolution())
	pr.SetCommonLabels(p.GetCommonLabels())

	pwr, pwok := p.(v1.PackageWithRuntime)
	prwr, prok := pr.(v1.PackageRevisionWithRuntime)
	if pwok && prok {
		prwr.SetRuntimeConfigRef(pwr.GetRuntimeConfigRef())
		prwr.SetControllerConfigRef(pwr.GetControllerConfigRef())
		prwr.SetTLSServerSecretName(pwr.GetTLSServerSecretName())
		prwr.SetTLSClientSecretName(pwr.GetTLSClientSecretName())
	}

	// If current revision is not active, and we have an automatic or
	// undefined activation policy, always activate.
	if pr.GetDesiredState() != v1.PackageRevisionActive && (p.GetActivationPolicy() == nil || *p.GetActivationPolicy() == v1.AutomaticActivation) {
		pr.SetDesiredState(v1.PackageRevisionActive)
	}

	controlRef := meta.AsController(meta.TypedReferenceTo(p, p.GetObjectKind().GroupVersionKind()))
	controlRef.BlockOwnerDeletion = ptr.To(true)
	meta.AddOwnerReference(pr, controlRef)
	if err := r.client.Apply(ctx, pr, resource.MustBeControllableBy(p.GetUID())); err != nil {
		if kerrors.IsConflict(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		err = errors.Wrap(err, errApplyPackageRevision)
		r.record.Event(p, event.Warning(reasonInstall, err))
		return reconcile.Result{}, err
	}

	// Handle changes in labels
	same := reflect.DeepEqual(pr.GetCommonLabels(), p.GetCommonLabels())
	if !same {
		pr.SetCommonLabels(p.GetCommonLabels())
		if err := r.client.Update(ctx, pr); err != nil {
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			err = errors.Wrap(err, errApplyPackageRevision)
			r.record.Event(p, event.Warning(reasonInstall, err))
			return reconcile.Result{}, err
		}
	}

	p.SetConditions(v1.Active())

	// If current revision is still not active, the package is inactive.
	if pr.GetDesiredState() != v1.PackageRevisionActive {
		p.SetConditions(v1.Inactive().WithMessage("Package is inactive"))
	}

	// NOTE(hasheddan): when the first package revision is created for a
	// package, the health of the package is not set until the revision reports
	// its health. If updating from an existing revision, the package health
	// will match the health of the old revision until the next reconcile.
	return pullBasedRequeue(p.GetPackagePullPolicy()), errors.Wrap(r.client.Status().Update(ctx, p), errUpdateStatus)
}
