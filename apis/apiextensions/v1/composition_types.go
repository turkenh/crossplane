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
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
)

const (
	errMathNoMultiplier       = "no input is given"
	errMathInputNonNumber     = "input is required to be a number for math transformer"
	errMultipleValuesReturned = "multiple output values returned - please add a combine transform"
	errPatchSetType           = "a patch in a PatchSet cannot be of type PatchSet"

	errFmtConfigMissing                = "given type %s requires configuration"
	errFmtConvertInputTypeNotSupported = "input type %s is not supported"
	errFmtConversionPairNotSupported   = "conversion from %s to %s is not supported"
	errFmtInvalidPatchType             = "patch type %s is unsupported"
	errFmtMapTypeNotSupported          = "type %s is not supported for map transform"
	errFmtMapNotFound                  = "key %s is not found in map"
	errFmtRequiredField                = "%s is required by type %s"
	errFmtResolveAtIndex               = "could not resolve input value at index %d"
	errFmtTransformAtIndex             = "transform at index %d returned error"
	errFmtTransformTypeFailed          = "%s transform could not resolve"
	errFmtTypeNotSupported             = "transform type %s is not supported"
	errFmtUndefinedPatchSet            = "cannot find PatchSet by name %s"
)

// CompositionSpec specifies the desired state of the definition.
type CompositionSpec struct {
	// CompositeTypeRef specifies the type of composite resource that this
	// composition is compatible with.
	// +immutable
	CompositeTypeRef TypeReference `json:"compositeTypeRef"`

	// PatchSets define a named set of patches that may be included by
	// any resource in this Composition.
	// PatchSets cannot themselves refer to other PatchSets.
	// +optional
	PatchSets []PatchSet `json:"patchSets,omitempty"`

	// Resources is the list of resource templates that will be used when a
	// composite resource referring to this composition is created.
	Resources []ComposedTemplate `json:"resources"`

	// WriteConnectionSecretsToNamespace specifies the namespace in which the
	// connection secrets of composite resource dynamically provisioned using
	// this composition will be created.
	// +optional
	WriteConnectionSecretsToNamespace *string `json:"writeConnectionSecretsToNamespace,omitempty"`
}

// InlinePatchSets dereferences PatchSets and includes their patches inline. The
// updated CompositionSpec should not be persisted to the API server.
func (cs *CompositionSpec) InlinePatchSets() error {
	pn := make(map[string][]Patch)
	for _, s := range cs.PatchSets {
		for _, p := range s.Patches {
			if p.Type == PatchTypePatchSet {
				return errors.New(errPatchSetType)
			}
		}
		pn[s.Name] = s.Patches
	}

	for i, r := range cs.Resources {
		po := []Patch{}
		for _, p := range r.Patches {
			if p.Type != PatchTypePatchSet {
				po = append(po, p)
				continue
			}
			if p.PatchSetName == nil {
				return errors.Errorf(errFmtRequiredField, "PatchSetName", p.Type)
			}
			ps, ok := pn[*p.PatchSetName]
			if !ok {
				return errors.Errorf(errFmtUndefinedPatchSet, *p.PatchSetName)
			}
			po = append(po, ps...)
		}
		cs.Resources[i].Patches = po
	}
	return nil
}

// A PatchSet is a set of patches that can be reused from all resources within
// a Composition.
type PatchSet struct {
	// Name of this PatchSet.
	Name string `json:"name"`

	// Patches will be applied as an overlay to the base resource.
	Patches []Patch `json:"patches"`
}

// TypeReference is used to refer to a type for declaring compatibility.
type TypeReference struct {
	// APIVersion of the type.
	APIVersion string `json:"apiVersion"`

	// Kind of the type.
	Kind string `json:"kind"`
}

// TypeReferenceTo returns a reference to the supplied GroupVersionKind
func TypeReferenceTo(gvk schema.GroupVersionKind) TypeReference {
	return TypeReference{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind}
}

// ComposedTemplate is used to provide information about how the composed resource
// should be processed.
type ComposedTemplate struct {
	// Base is the target resource that the patches will be applied on.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Base runtime.RawExtension `json:"base"`

	// Patches will be applied as overlay to the base resource.
	// +optional
	Patches []Patch `json:"patches,omitempty"`

	// ConnectionDetails lists the propagation secret keys from this target
	// resource to the composition instance connection secret.
	// +optional
	ConnectionDetails []ConnectionDetail `json:"connectionDetails,omitempty"`

	// ReadinessChecks allows users to define custom readiness checks. All checks
	// have to return true in order for resource to be considered ready. The
	// default readiness check is to have the "Ready" condition to be "True".
	// +optional
	ReadinessChecks []ReadinessCheck `json:"readinessChecks,omitempty"`
}

// TypeReadinessCheck is used for readiness check types.
type TypeReadinessCheck string

// The possible values for readiness check type.
const (
	ReadinessCheckNonEmpty     TypeReadinessCheck = "NonEmpty"
	ReadinessCheckMatchString  TypeReadinessCheck = "MatchString"
	ReadinessCheckMatchInteger TypeReadinessCheck = "MatchInteger"
	ReadinessCheckNone         TypeReadinessCheck = "None"
)

// ReadinessCheck is used to indicate how to tell whether a resource is ready
// for consumption
type ReadinessCheck struct {
	// Type indicates the type of probe you'd like to use.
	// +kubebuilder:validation:Enum="MatchString";"MatchInteger";"NonEmpty";"None"
	Type TypeReadinessCheck `json:"type"`

	// FieldPath shows the path of the field whose value will be used.
	// +optional
	FieldPath string `json:"fieldPath,omitempty"`

	// MatchString is the value you'd like to match if you're using "MatchString" type.
	// +optional
	MatchString string `json:"matchString,omitempty"`

	// MatchInt is the value you'd like to match if you're using "MatchInt" type.
	// +optional
	MatchInteger int64 `json:"matchInteger,omitempty"`
}

// A PatchType is a type of patch.
type PatchType string

// Patch types.
const (
	PatchTypeFromCompositeFieldPath          PatchType = "FromCompositeFieldPath" // Default
	PatchTypeFromMultipleCompositeFieldPaths PatchType = "FromMultipleCompositeFieldPaths"
	PatchTypePatchSet                        PatchType = "PatchSet"
	PatchTypeToCompositeFieldPath            PatchType = "ToCompositeFieldPath"
)

// Patch objects are applied between composite and composed resources. Their
// behaviour depends on the Type selected. The default Type,
// FromCompositeFieldPath, copies a value from the composite resource to
// the composed resource, applying any defined transformers.
type Patch struct {
	// Type sets the patching behaviour to be used. Each patch type may require
	// its' own fields to be set on the Patch object.
	// +optional
	// +kubebuilder:validation:Enum=FromCompositeFieldPath;PatchSet;ToCompositeFieldPath;FromMultipleCompositeFieldPaths
	// +kubebuilder:default=FromCompositeFieldPath
	Type PatchType `json:"type,omitempty"`

	// FromFieldPath is the path of the field on the composed resource whose value
	// to be used as input. Required when type is FromCompositeFieldPath.
	// +optional
	FromFieldPath *string `json:"fromFieldPath,omitempty"`

	// FromMultipleFieldPaths is a list of paths to fields on the upstream resource whose
	// values are used as input. Required when type is FromMultipleCompositeFieldPaths.
	// +optional
	FromMultipleFieldPaths []string `json:"fromMultipleFieldPaths,omitempty"`

	// ToFieldPath is the path of the field on the base resource whose value will
	// be changed with the result of transforms. Leave empty if you'd like to
	// propagate to the same path on the target resource.
	// +optional
	ToFieldPath *string `json:"toFieldPath,omitempty"`

	// PatchSetName to include patches from. Required when type is PatchSet.
	// +optional
	PatchSetName *string `json:"patchSetName,omitempty"`

	// Transforms are the list of functions that are used as a FIFO pipe for the
	// input to be transformed.
	// +optional
	Transforms []Transform `json:"transforms,omitempty"`
}

// Apply executes a patching operation between the from and to resources.
// Applies all patch types unless an 'only' filter is supplied.
func (c *Patch) Apply(from, to runtime.Object, only ...PatchType) error {
	if c.filterPatch(only...) {
		return nil
	}

	switch c.Type {
	case PatchTypeFromCompositeFieldPath:
		return c.applyFromCompositeFieldPatch(from, to)
	case PatchTypeFromMultipleCompositeFieldPaths:
		return c.applyFromMultipleCompositeFieldsPatch(from, to)
	case PatchTypeToCompositeFieldPath:
		return c.applyFromCompositeFieldPatch(to, from)
	case PatchTypePatchSet:
		// Already resolved - nothing to do.
	}
	return errors.Errorf(errFmtInvalidPatchType, c.Type)
}

// filterPatch returns true if patch should be filtered (not applied)
func (c *Patch) filterPatch(only ...PatchType) bool {
	// filter does not apply if not set
	if len(only) == 0 {
		return false
	}

	for _, patchType := range only {
		if patchType == c.Type {
			return false
		}
	}
	return true
}

// applyFromFieldPathPatch patches the "to" resource, using a source field
// on the "from" resource. Values may be transformed if any are defined on
// applyTransforms applies a list of transforms to patch value(s).
// The transform chain must return a single value. If it returns
// multiple values, error and prompt the user to add a combine transform.
func (c *Patch) applyTransforms(input []interface{}) (interface{}, error) {
	var err error
	for i, t := range c.Transforms {
		if input, err = t.Transform(input); err != nil {
			return nil, errors.Wrapf(err, errFmtTransformAtIndex, i)
		}
	}
	// We can only patch a single output value, so if we still have
	// multiple values here, then the user has not combined the input
	// values.
	if len(input) > 1 {
		return nil, errors.New(errMultipleValuesReturned)
	}
	return input[0], nil
}

// applyFromCompositeFieldPatch patches the composed resource, using a source field
// on the composite resource. Values may be transformed if any are defined on
// the patch.
func (c *Patch) applyFromCompositeFieldPatch(from, to runtime.Object) error { // nolint:gocyclo
	// NOTE(benagricola): The cyclomatic complexity here is from error checking
	// at each stage of the patching process, in addition to Apply methods now
	// being responsible for checking the validity of their input fields
	// (necessary because with multiple patch types, the input fields
	// must be +optional).
	if c.FromFieldPath == nil {
		return errors.Errorf(errFmtRequiredField, "FromFieldPath", c.Type)
	}

	// Default to patching the same field on the composed resource.
	if c.ToFieldPath == nil {
		c.ToFieldPath = c.FromFieldPath
	}

	fromMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(from)
	if err != nil {
		return err
	}

	in, err := fieldpath.Pave(fromMap).GetValue(*c.FromFieldPath)
	if fieldpath.IsNotFound(err) {
		// A composition may want to opportunistically patch from a field path
		// that may or may not exist in the composite, for example by patching
		// {fromFieldPath: metadata.labels, toFieldPath: metadata.labels}. We
		// don't consider a reference to a non-existent path to be an issue; if
		// the relevant toFieldPath is required by the composed resource we'll
		// report that fact when we attempt to reconcile the composite.
		return nil
	}
	if err != nil {
		return err
	}

	// Apply transform pipeline
	out, err := c.applyTransforms([]interface{}{in})
	if err != nil {
		return err
	}

	if u, ok := to.(interface{ UnstructuredContent() map[string]interface{} }); ok {
		return fieldpath.Pave(u.UnstructuredContent()).SetValue(*c.ToFieldPath, out)
	}

	toMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(to)
	if err != nil {
		return err
	}
	if err := fieldpath.Pave(toMap).SetValue(*c.ToFieldPath, out); err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(toMap, to)
}

// applyFromMultipleCompositeFieldsPatch patches the composed resource, using a list of
// source fields on the composite resource. Without a transform, values are turned into their
// string representation and concatenated. Use a string transform with multiple placeholders
// to control the format more explicitly.
func (c *Patch) applyFromMultipleCompositeFieldsPatch(from, to runtime.Object) error { // nolint:gocyclo
	if len(c.FromMultipleFieldPaths) == 0 {
		return errors.Errorf(errFmtRequiredField, "FromMultipleFieldPaths", c.Type)
	}

	// Unlike FromCompositeFieldPath, defaulting the To field to the From field might
	// introduce some confusion, as we can specify multiple From fields. Easier on the
	// brain to just make this explicit.
	if c.ToFieldPath == nil {
		return errors.Errorf(errFmtRequiredField, "ToFieldPath", c.Type)
	}

	fromMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(from)
	if err != nil {
		return err
	}

	in := make([]interface{}, len(c.FromMultipleFieldPaths))

	// Get value of each source field, or error
	for i, sp := range c.FromMultipleFieldPaths {
		iv, err := fieldpath.Pave(fromMap).GetValue(sp)

		// If field is not found, do not patch.
		if fieldpath.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}
		in[i] = iv
	}

	// Apply transforms pipeline
	out, err := c.applyTransforms(in)
	if err != nil {
		return err
	}

	if u, ok := to.(interface{ UnstructuredContent() map[string]interface{} }); ok {
		return fieldpath.Pave(u.UnstructuredContent()).SetValue(*c.ToFieldPath, out)
	}

	toMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(to)
	if err != nil {
		return err
	}
	if err := fieldpath.Pave(toMap).SetValue(*c.ToFieldPath, out); err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(toMap, to)
}

// TransformType is type of the transform function to be chosen.
type TransformType string

// Accepted TransformTypes.
const (
	TransformTypeMap     TransformType = "map"
	TransformTypeMath    TransformType = "math"
	TransformTypeString  TransformType = "string"
	TransformTypeConvert TransformType = "convert"
	TransformTypeCombine TransformType = "combine"
)

// Transform is a unit of process whose input is transformed into an output with
// the supplied configuration.
type Transform struct {

	// Type of the transform to be run.
	Type TransformType `json:"type"`

	// Math is used to transform the input via mathematical operations such as
	// multiplication.
	// +optional
	Math *MathTransform `json:"math,omitempty"`

	// Map uses the input as a key in the given map and returns the value.
	// +optional
	Map *MapTransform `json:"map,omitempty"`

	// String is used to transform the input into a string or a different kind
	// of string. Note that the input does not necessarily need to be a string.
	// +optional
	String *StringTransform `json:"string,omitempty"`

	// Convert is used to cast the input into the given output type.
	// +optional
	Convert *ConvertTransform `json:"convert,omitempty"`

	// Combine is used to turn multiple input values into a single
	// output value. When using a PatchType that takes multiple input
	// values, a combine transform must be used to turn it into a single
	// output value.
	// +optional
	Combine *CombineTransform `json:"combine,omitempty"`
}

// Transform calls the appropriate Transformer.
func (t *Transform) Transform(input []interface{}) ([]interface{}, error) {
	var transformer interface {
		Resolve(input []interface{}) ([]interface{}, error)
	}

	switch t.Type {
	case TransformTypeMath:
		transformer = t.Math
	case TransformTypeMap:
		transformer = t.Map
	case TransformTypeString:
		transformer = t.String
	case TransformTypeConvert:
		transformer = t.Convert
	case TransformTypeCombine:
		transformer = t.Combine
	default:
		return nil, errors.Errorf(errFmtTypeNotSupported, string(t.Type))
	}
	// An interface equals nil only if both the type and value are nil. Above,
	// even if t.<Type> is nil, its type is assigned to "transformer" but we're
	// interested in whether only the value is nil or not.
	if reflect.ValueOf(transformer).IsNil() {
		return nil, errors.Errorf(errFmtConfigMissing, string(t.Type))
	}
	out, err := transformer.Resolve(input)
	return out, errors.Wrapf(err, errFmtTransformTypeFailed, string(t.Type))
}

// resolverFunc represents a function that can resolve a single
// input value into a single output value.
type resolverFunc func(input interface{}) (interface{}, error)

// resolveMultiple executes a resolverFunc on a list of inputs,
// returning a list of outputs with length identical to that of inputs.
// This function is likely to be duplicated into every Resolve method so
// it was extracted, allowing each Resolve method to call this with their
// own resolverFunc implementation.
func resolveMultiple(input []interface{}, resolver resolverFunc) ([]interface{}, error) {
	il := len(input)
	out := make([]interface{}, il)

	// If we're only passed one value, call the resolverFunc
	// directly. Don't wrap any errors as the index of the
	// input doesn't matter.
	if il < 2 {
		rv, err := resolver(input[0])
		if err != nil {
			return nil, err
		}
		out[0] = rv
		return out, nil
	}

	// With multiple values - loop over each value, and resolve it
	// against the given resolverFunc
	for i, v := range input {
		rv, err := resolver(v)
		if err != nil {
			return nil, errors.Wrapf(err, errFmtResolveAtIndex, i)
		}
		out[i] = rv
	}
	return out, nil
}

// MathTransform conducts mathematical operations on the input with the given
// configuration in its properties.
type MathTransform struct {
	// Multiply the value.
	// +optional
	Multiply *int64 `json:"multiply,omitempty"`
}

// Resolve runs the Math transform.
func (m *MathTransform) Resolve(input []interface{}) ([]interface{}, error) {
	if m.Multiply == nil {
		return nil, errors.New(errMathNoMultiplier)
	}

	return resolveMultiple(input, m.resolveOne)
}

// resolveOne resolves a single Math value
func (m *MathTransform) resolveOne(input interface{}) (interface{}, error) {
	switch i := input.(type) {
	case int64:
		return *m.Multiply * i, nil
	case int:
		return *m.Multiply * int64(i), nil
	default:
		return nil, errors.New(errMathInputNonNumber)
	}
}

// MapTransform returns a value for the input from the given map.
type MapTransform struct {
	// TODO(negz): Are Pairs really optional if a MapTransform was specified?

	// Pairs is the map that will be used for transform.
	// +optional
	Pairs map[string]string `json:",inline"`
}

// NOTE(negz): The Kubernetes JSON decoder doesn't seem to like inlining a map
// into a struct - doing so results in a seemingly successful unmarshal of the
// data, but an empty map. We must keep the ,inline tag nevertheless in order to
// trick the CRD generator into thinking MapTransform is an arbitrary map (i.e.
// generating a validation schema with string additionalProperties), but the
// actual marshalling is handled by the marshal methods below.

// UnmarshalJSON into this MapTransform.
func (m *MapTransform) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &m.Pairs)
}

// MarshalJSON from this MapTransform.
func (m MapTransform) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Pairs)
}

// Resolve runs the Map transform.
func (m *MapTransform) Resolve(input []interface{}) ([]interface{}, error) {
	return resolveMultiple(input, m.resolveOne)
}

// resolveOne resolves a single Map value
func (m *MapTransform) resolveOne(input interface{}) (interface{}, error) {
	switch i := input.(type) {
	case string:
		val, ok := m.Pairs[i]
		if !ok {
			return nil, errors.Errorf(errFmtMapNotFound, i)
		}
		return val, nil
	default:
		return nil, errors.Errorf(errFmtMapTypeNotSupported, reflect.TypeOf(input).String())
	}
}

// A StringTransform returns a string given the supplied input.
type StringTransform struct {
	// Format the input using a Go format string. See
	// https://golang.org/pkg/fmt/ for details.
	Format string `json:"fmt"`
}

// Resolve runs the Map transform.
func (s *StringTransform) Resolve(input []interface{}) ([]interface{}, error) {
	return resolveMultiple(input, s.resolveOne)
}

// resolveOne resolves a single String value
func (s *StringTransform) resolveOne(input interface{}) (interface{}, error) {
	return fmt.Sprintf(s.Format, input), nil
}

// The list of supported ConvertTransform input and output types.
const (
	ConvertTransformTypeString  = "string"
	ConvertTransformTypeBool    = "bool"
	ConvertTransformTypeInt     = "int"
	ConvertTransformTypeFloat64 = "float64"
)

type conversionPair struct {
	From string
	To   string
}

var conversions = map[conversionPair]func(interface{}) (interface{}, error){
	{From: ConvertTransformTypeString, To: ConvertTransformTypeInt}: func(i interface{}) (interface{}, error) {
		return strconv.Atoi(i.(string))
	},
	{From: ConvertTransformTypeString, To: ConvertTransformTypeBool}: func(i interface{}) (interface{}, error) {
		return strconv.ParseBool(i.(string))
	},
	{From: ConvertTransformTypeString, To: ConvertTransformTypeFloat64}: func(i interface{}) (interface{}, error) {
		return strconv.ParseFloat(i.(string), 64)
	},

	{From: ConvertTransformTypeInt, To: ConvertTransformTypeString}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return strconv.Itoa(i.(int)), nil
	},
	{From: ConvertTransformTypeInt, To: ConvertTransformTypeBool}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return i.(int) == 1, nil
	},
	{From: ConvertTransformTypeInt, To: ConvertTransformTypeFloat64}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return float64(i.(int)), nil
	},

	{From: ConvertTransformTypeBool, To: ConvertTransformTypeString}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return strconv.FormatBool(i.(bool)), nil
	},
	{From: ConvertTransformTypeBool, To: ConvertTransformTypeInt}: func(i interface{}) (interface{}, error) { // nolint:unparam
		if i.(bool) {
			return 1, nil
		}
		return 0, nil
	},
	{From: ConvertTransformTypeBool, To: ConvertTransformTypeFloat64}: func(i interface{}) (interface{}, error) { // nolint:unparam
		if i.(bool) {
			return float64(1), nil
		}
		return float64(0), nil
	},

	{From: ConvertTransformTypeFloat64, To: ConvertTransformTypeString}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return strconv.FormatFloat(i.(float64), 'f', -1, 64), nil
	},
	{From: ConvertTransformTypeFloat64, To: ConvertTransformTypeInt}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return int(i.(float64)), nil
	},
	{From: ConvertTransformTypeFloat64, To: ConvertTransformTypeBool}: func(i interface{}) (interface{}, error) { // nolint:unparam
		return i.(float64) == float64(1), nil
	},
}

// A ConvertTransform converts the input into a new object whose type is supplied.
type ConvertTransform struct {
	// ToType is the type of the output of this transform.
	// +kubebuilder:validation:Enum=string;int;bool;float64
	ToType string `json:"toType"`
}

// Resolve runs the Convert transform.
func (s *ConvertTransform) Resolve(input []interface{}) ([]interface{}, error) {
	return resolveMultiple(input, s.resolveOne)
}

// resolveOne resolves a single Convert value
func (s *ConvertTransform) resolveOne(input interface{}) (interface{}, error) {
	switch reflect.TypeOf(input).Kind().String() {
	case s.ToType:
		return input, nil
	case ConvertTransformTypeString, ConvertTransformTypeBool, ConvertTransformTypeInt, ConvertTransformTypeFloat64:
		break
	default:
		return nil, errors.Errorf(errFmtConvertInputTypeNotSupported, reflect.TypeOf(input).Kind().String())
	}
	f, ok := conversions[conversionPair{From: reflect.TypeOf(input).Kind().String(), To: s.ToType}]
	if !ok {
		return nil, errors.Errorf(errFmtConversionPairNotSupported, reflect.TypeOf(input).Kind().String(), s.ToType)
	}
	return f(input)
}

// A ConnectionDetailType is a type of connection detail.
type ConnectionDetailType string

// ConnectionDetailType types.
const (
	ConnectionDetailFromConnectionSecretKey ConnectionDetailType = "FromConnectionSecretKey" // Default
	ConnectionDetailFromFieldPath           ConnectionDetailType = "FromFieldPath"
	ConnectionDetailValue                   ConnectionDetailType = "Value"
)

// CombineTransformType is type of the combine transform function to be chosen.
type CombineTransformType string

// Accepted CombineTransformTypes.
const (
	CombineTransformTypeString CombineTransformType = "string"
)

// A CombineTransform combines multiple inputs to a single output
// using another transform.
type CombineTransform struct {
	// Format the input using a Go format string. See
	// https://golang.org/pkg/fmt/ for details.
	String *StringCombine `json:"string,omitempty"`

	// Type of the combine to be run.
	// +kubebuilder:validation:Enum=string
	Type CombineTransformType `json:"type"`
}

// Resolve calls the appropriate Combiner.
func (t *CombineTransform) Resolve(input []interface{}) ([]interface{}, error) {
	var combiner interface {
		Combine(input []interface{}) (interface{}, error)
	}

	switch t.Type {
	case CombineTransformTypeString:
		combiner = t.String
	default:
		return nil, errors.Errorf(errFmtTypeNotSupported, string(t.Type))
	}
	if reflect.ValueOf(combiner).IsNil() {
		return nil, errors.Errorf(errFmtConfigMissing, string(t.Type))
	}
	out, err := combiner.Combine(input)
	return []interface{}{out}, errors.Wrapf(err, errFmtTransformTypeFailed, string(t.Type))
}

// A StringCombine returns a single string given multiple inputs.
type StringCombine struct {
	// Format the input using a Go format string. See
	// https://golang.org/pkg/fmt/ for details.
	Format string `json:"fmt"`
}

// Combine runs the String combine.
func (s *StringCombine) Combine(input []interface{}) (interface{}, error) {
	return fmt.Sprintf(s.Format, input...), nil
}

// ConnectionDetail includes the information about the propagation of the connection
// information from one secret to another.
type ConnectionDetail struct {
	// Name of the connection secret key that will be propagated to the
	// connection secret of the composition instance. Leave empty if you'd like
	// to use the same key name.
	// +optional
	Name *string `json:"name,omitempty"`

	// Type sets the connection detail fetching behaviour to be used. Each connection detail type may require
	// its' own fields to be set on the ConnectionDetail object.
	// +optional
	// +kubebuilder:validation:Enum=FromConnectionSecretKey;FromFieldPath;Value
	// +kubebuilder:default=FromConnectionSecretKey
	Type ConnectionDetailType `json:"type,omitempty"`

	// FromConnectionSecretKey is the key that will be used to fetch the value
	// from the given target resource's secret
	// +optional
	FromConnectionSecretKey *string `json:"fromConnectionSecretKey,omitempty"`

	// FromFieldPath is the path of the field on the composed resource whose value
	// to be used as input. Name must be specified if the type is FromFieldPath is specified.
	// +optional
	FromFieldPath *string `json:"fromFieldPath,omitempty"`

	// Value that will be propagated to the connection secret of the composition
	// instance. Typically you should use FromConnectionSecretKey instead, but
	// an explicit value may be set to inject a fixed, non-sensitive connection
	// secret values, for example a well-known port. Supercedes
	// FromConnectionSecretKey when set.
	// +optional
	Value *string `json:"value,omitempty"`
}

// CompositionStatus shows the observed state of the composition.
type CompositionStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +genclient
// +genclient:nonNamespaced

// Composition defines the group of resources to be created when a compatible
// type is created with reference to the composition.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=crossplane
type Composition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompositionSpec   `json:"spec,omitempty"`
	Status CompositionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CompositionList contains a list of Compositions.
type CompositionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Composition `json:"items"`
}
