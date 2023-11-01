/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
*/

package composite

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/resource/unstructured/composed"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/crossplane/crossplane/internal/xcrd"
)

func TestRenderFromJSON(t *testing.T) {
	errInvalidChar := json.Unmarshal([]byte("olala"), &fake.Composed{})

	type args struct {
		o    resource.Object
		data []byte
	}
	type want struct {
		o   resource.Object
		err error
	}
	cases := map[string]struct {
		reason string
		args
		want
	}{
		"InvalidData": {
			reason: "We should return an error if the data can't be unmarshalled",
			args: args{
				o:    &fake.Composed{},
				data: []byte("olala"),
			},
			want: want{
				o:   &fake.Composed{},
				err: errors.Wrap(errInvalidChar, errUnmarshalJSON),
			},
		},
		"ExistingGVKChanged": {
			reason: "We should return an error if unmarshalling the base template changed the composed resource's group, version, or kind",
			args: args{
				o: composed.New(composed.FromReference(corev1.ObjectReference{
					APIVersion: "example.org/v1",
					Kind:       "Potato",
				})),
				data: []byte(`{"apiVersion": "example.org/v1", "kind": "Different"}`),
			},
			want: want{
				o: composed.New(composed.FromReference(corev1.ObjectReference{
					APIVersion: "example.org/v1",
					Kind:       "Different",
				})),
				err: errors.Errorf(errFmtKindChanged, "example.org/v1, Kind=Potato", "example.org/v1, Kind=Different"),
			},
		},
		"NewComposedResource": {
			reason: "A valid base template should apply successfully to a new (empty) composed resource",
			args: args{
				o:    composed.New(),
				data: []byte(`{"apiVersion": "example.org/v1", "kind": "Potato", "spec": {"cool": true}}`),
			},
			want: want{
				o: &composed.Unstructured{Unstructured: unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1",
						"kind":       "Potato",
						"spec": map[string]any{
							"cool": true,
						},
					},
				}},
			},
		},
		"ExistingComposedResource": {
			reason: "A valid base template should apply successfully to a new (empty) composed resource",
			args: args{
				o: composed.New(composed.FromReference(corev1.ObjectReference{
					APIVersion: "example.org/v1",
					Kind:       "Potato",
					Name:       "ola-superrandom",
				})),
				data: []byte(`{"apiVersion": "example.org/v1", "kind": "Potato", "spec": {"cool": true}}`),
			},
			want: want{
				o: &composed.Unstructured{Unstructured: unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1",
						"kind":       "Potato",
						"metadata": map[string]any{
							"name": "ola-superrandom",
						},
						"spec": map[string]any{
							"cool": true,
						},
					},
				}},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := RenderFromJSON(tc.args.o, tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nRenderFromJSON(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, tc.args.o); diff != "" {
				t.Errorf("\n%s\nRenderFromJSON(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestRenderComposedResourceMetadata(t *testing.T) {
	controlled := &fake.Composed{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{{
				Controller: ptr.To(true),
				UID:        "very-random",
			}},
		},
	}
	errRef := meta.AddControllerReference(controlled, metav1.OwnerReference{UID: "not-very-random"})

	type args struct {
		xr resource.Composite
		cd resource.Composed
		rn ResourceName
	}
	type want struct {
		cd  resource.Composed
		err error
	}
	cases := map[string]struct {
		reason string
		args
		want
	}{
		"ConflictingControllerReference": {
			reason: "We should return an error if the composed resource has an existing (and different) controller reference",
			args: args{
				xr: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						UID: "somewhat-random",
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{{
							Controller: ptr.To(true),
							UID:        "very-random",
						}},
					},
				},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "prefix-",
						OwnerReferences: []metav1.OwnerReference{{
							Controller: ptr.To(true),
							UID:        "very-random",
						}},
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
				err: errors.Wrap(errRef, errSetControllerRef),
			},
		},
		"CompatibleControllerReference": {
			reason: "We should not return an error if the composed resource has an existing (and matching) controller reference",
			args: args{
				xr: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						UID: "somewhat-random",
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{{
							Controller: ptr.To(true),
							UID:        "somewhat-random",
						}},
					},
				},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "prefix-",
						OwnerReferences: []metav1.OwnerReference{{
							Controller:         ptr.To(true),
							BlockOwnerDeletion: ptr.To(true),
							UID:                "somewhat-random",
						}},
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
			},
		},
		"NoControllerReference": {
			reason: "We should not return an error if the composed resource has no controller reference",
			args: args{
				xr: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cool-xr",
						UID:  "somewhat-random",
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
				cd: &fake.Composed{},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "prefix-",
						OwnerReferences: []metav1.OwnerReference{{
							Controller:         ptr.To(true),
							BlockOwnerDeletion: ptr.To(true),
							UID:                "somewhat-random",
							Name:               "cool-xr",
						}},
						Labels: map[string]string{
							xcrd.LabelKeyNamePrefixForComposed: "prefix",
							xcrd.LabelKeyClaimName:             "name",
							xcrd.LabelKeyClaimNamespace:        "namespace",
						},
					},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := RenderComposedResourceMetadata(tc.args.cd, tc.args.xr, tc.args.rn)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nRenderComposedResourceMetadata(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.cd, tc.args.cd); diff != "" {
				t.Errorf("\n%s\nRenderComposedResourceMetadata(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGenerateName(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		ctx context.Context
		cd  resource.Composed
	}
	type want struct {
		cd  resource.Composed
		err error
	}
	cases := map[string]struct {
		reason string
		client client.Client
		args
		want
	}{
		"SkipGenerateNamedResources": {
			reason: "We should not try naming a resource that already have a name",
			// We must be returning early, or else we'd hit this error.
			client: &test.MockClient{MockCreate: test.NewMockCreateFn(errBoom)},
			args: args{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					Name: "already-has-a-cool-name",
				}},
			},
			want: want{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					Name: "already-has-a-cool-name",
				}},
				err: nil,
			},
		},
		"SkipGenerateNameForResourcesWithoutGenerateName": {
			reason: "We should not try to name resources that don't have a generate name (though that should never happen)",
			// We must be returning early, or else we'd hit this error.
			client: &test.MockClient{MockCreate: test.NewMockCreateFn(errBoom)},
			args: args{
				cd: &fake.Composed{}, // Conspicously missing a generate name.
			},
			want: want{
				cd:  &fake.Composed{},
				err: nil,
			},
		},
		"NameGeneratorClientError": {
			reason: "Client error finding a free name for a composed resource",
			client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
			args: args{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
			},
			want: want{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
				err: errBoom,
			},
		},
		"Success": {
			reason: "Name is found on first try",
			client: &test.MockClient{MockGet: test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{Resource: "CoolResource"}, "cool-resource-42"))},
			args: args{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
			},
			want: want{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
					Name:         "cool-resource-42",
				}},
			},
		},
		"SuccessAfterConflict": {
			reason: "Name is found on second try",
			client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				if key.Name == "cool-resource-42" {
					return nil
				}
				return kerrors.NewNotFound(schema.GroupResource{Resource: "CoolResource"}, key.Name)
			}},
			args: args{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
			},
			want: want{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
					Name:         "cool-resource-43",
				}},
			},
		},
		"AlwaysConflict": {
			reason: "Name cannot be found",
			client: &test.MockClient{MockGet: test.NewMockGetFn(nil)},
			args: args{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
			},
			want: want{
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{
					GenerateName: "cool-resource-",
				}},
				err: errors.New(errGenerateName),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewAPINameGenerator(tc.client)
			r.namer = &mockNameGenerator{last: 41}
			err := r.GenerateName(tc.args.ctx, tc.args.cd)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nDryRunRender(...): -want, +got:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.cd, tc.args.cd); diff != "" {
				t.Errorf("\n%s\nDryRunRender(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

type mockNameGenerator struct {
	last int
}

func (m *mockNameGenerator) GenerateName(prefix string) string {
	m.last++
	return prefix + strconv.Itoa(m.last)
}
