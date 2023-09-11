//go:build !ignore_autogenerated

/*
Copyright 2021 The Crossplane Authors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dependency) DeepCopyInto(out *Dependency) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dependency.
func (in *Dependency) DeepCopy() *Dependency {
	if in == nil {
		return nil
	}
	out := new(Dependency)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Lock) DeepCopyInto(out *Lock) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = make([]LockPackage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Lock.
func (in *Lock) DeepCopy() *Lock {
	if in == nil {
		return nil
	}
	out := new(Lock)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Lock) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LockList) DeepCopyInto(out *LockList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Lock, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LockList.
func (in *LockList) DeepCopy() *LockList {
	if in == nil {
		return nil
	}
	out := new(LockList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LockList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LockPackage) DeepCopyInto(out *LockPackage) {
	*out = *in
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]Dependency, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LockPackage.
func (in *LockPackage) DeepCopy() *LockPackage {
	if in == nil {
		return nil
	}
	out := new(LockPackage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageRevisionSpec) DeepCopyInto(out *PackageRevisionSpec) {
	*out = *in
	if in.ControllerConfigReference != nil {
		in, out := &in.ControllerConfigReference, &out.ControllerConfigReference
		*out = new(v1.Reference)
		(*in).DeepCopyInto(*out)
	}
	if in.PackagePullSecrets != nil {
		in, out := &in.PackagePullSecrets, &out.PackagePullSecrets
		*out = make([]corev1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.PackagePullPolicy != nil {
		in, out := &in.PackagePullPolicy, &out.PackagePullPolicy
		*out = new(corev1.PullPolicy)
		**out = **in
	}
	if in.IgnoreCrossplaneConstraints != nil {
		in, out := &in.IgnoreCrossplaneConstraints, &out.IgnoreCrossplaneConstraints
		*out = new(bool)
		**out = **in
	}
	if in.SkipDependencyResolution != nil {
		in, out := &in.SkipDependencyResolution, &out.SkipDependencyResolution
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageRevisionSpec.
func (in *PackageRevisionSpec) DeepCopy() *PackageRevisionSpec {
	if in == nil {
		return nil
	}
	out := new(PackageRevisionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageRevisionStatus) DeepCopyInto(out *PackageRevisionStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	in.ControllerRef.DeepCopyInto(&out.ControllerRef)
	if in.ObjectRefs != nil {
		in, out := &in.ObjectRefs, &out.ObjectRefs
		*out = make([]v1.TypedReference, len(*in))
		copy(*out, *in)
	}
	if in.PermissionRequests != nil {
		in, out := &in.PermissionRequests, &out.PermissionRequests
		*out = make([]rbacv1.PolicyRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageRevisionStatus.
func (in *PackageRevisionStatus) DeepCopy() *PackageRevisionStatus {
	if in == nil {
		return nil
	}
	out := new(PackageRevisionStatus)
	in.DeepCopyInto(out)
	return out
}
