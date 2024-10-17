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

package v1

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// Condition types.
const (
	// A TypeInstalled indicates whether a package has been installed.
	TypeInstalled xpv1.ConditionType = "Installed"

	// A TypeHealthy indicates whether a package is healthy.
	TypeHealthy xpv1.ConditionType = "Healthy"

	// A TypeSignatureVerified indicates whether a package's signature
	// has been verified.
	TypeSignatureVerified xpv1.ConditionType = "SignatureVerified"
)

// Reasons a package is or is not installed.
const (
	ReasonUnpacking     xpv1.ConditionReason = "UnpackingPackage"
	ReasonInactive      xpv1.ConditionReason = "InactivePackageRevision"
	ReasonActive        xpv1.ConditionReason = "ActivePackageRevision"
	ReasonUnhealthy     xpv1.ConditionReason = "UnhealthyPackageRevision"
	ReasonHealthy       xpv1.ConditionReason = "HealthyPackageRevision"
	ReasonUnknownHealth xpv1.ConditionReason = "UnknownPackageRevisionHealth"
)

// Reasons a package's signature is or is not verified.
const (
	ReasonVerificationSucceeded xpv1.ConditionReason = "VerificationSucceeded"
	ReasonVerificationFailed    xpv1.ConditionReason = "VerificationFailed"
	ReasonVerificationSkipped   xpv1.ConditionReason = "VerificationSkipped"
)

// Unpacking indicates that the package manager is waiting for a package
// revision to be unpacked.
func Unpacking() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeInstalled,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUnpacking,
	}
}

// Inactive indicates that the package manager is waiting for a package
// revision to be transitioned to an active state.
func Inactive() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeInstalled,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonInactive,
	}
}

// Active indicates that the package manager has installed and activated
// a package revision.
func Active() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeInstalled,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonActive,
	}
}

// Unhealthy indicates that the current revision is unhealthy.
func Unhealthy() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeHealthy,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUnhealthy,
	}
}

// Healthy indicates that the current revision is healthy.
func Healthy() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeHealthy,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonHealthy,
	}
}

// UnknownHealth indicates that the health of the current revision is unknown.
func UnknownHealth() xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeHealthy,
		Status:             corev1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUnknownHealth,
	}
}

// VerificationSucceededWith returns a condition indicating that a package's
// signature has been successfully verified using the supplied image config.
func VerificationSucceededWith(imageConfig string) xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeSignatureVerified,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonVerificationSucceeded,
		Message:            fmt.Sprintf("Signature verification succeeded with ImageConfig named %q", imageConfig),
	}
}

// VerificationFailedWith returns a condition indicating that a package's
// signature verification failed using the supplied image config.
func VerificationFailedWith(imageConfig string, err error) xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeSignatureVerified,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonVerificationFailed,
		Message:            fmt.Sprintf("Signature verification failed with ImageConfig named %q: %s", imageConfig, err),
	}
}

// VerificationSkipped returns a condition indicating that signature
// verification was skipped for a package.
func VerificationSkipped() xpv1.Condition {
	return xpv1.Condition{
		Type: TypeSignatureVerified,
		// TODO: This feels a bit weird, skipped but true.
		//  Should we be using ConditionFalse here? Should package revision
		//  reconciler wait for the condition to exist or to be true before
		//  proceeding?
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonVerificationSkipped,
	}
}
