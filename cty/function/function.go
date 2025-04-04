package function

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
)

// Function represents a function. This is the main type in this package.
type Function struct {
	spec *Spec
}

// Spec is the specification of a function, used to instantiate
// a new Function.
type Spec struct {
	// Params is a description of the positional parameters for the function.
	// The standard checking logic rejects any calls that do not provide
	// arguments conforming to this definition, freeing the function
	// implementer from dealing with such inconsistencies.
	Params []Parameter

	// VarParam is an optional specification of additional "varargs" the
	// function accepts. If this is non-nil then callers may provide an
	// arbitrary number of additional arguments (after those matching with
	// the fixed parameters in Params) that conform to the given specification,
	// which will appear as additional values in the slices of values
	// provided to the type and implementation functions.
	VarParam *Parameter

	// Type is the TypeFunc that decides the return type of the function
	// given its arguments, which may be Unknown. See the documentation
	// of TypeFunc for more information.
	//
	// Use StaticReturnType if the function's return type does not vary
	// depending on its arguments.
	Type TypeFunc

	// Impl is the ImplFunc that implements the function's behavior.
	//
	// Functions are expected to behave as pure functions, and not create
	// any visible side-effects.
	//
	// If a TypeFunc is also provided, the value returned from Impl *must*
	// conform to the type it returns, or a call to the function will panic.
	Impl ImplFunc
}

// New creates a new function with the given specification.
//
// After passing a Spec to this function, the caller must no longer read from
// or mutate it.
func New(spec *Spec) Function {
	f := Function{
		spec: spec,
	}
	return f
}

// TypeFunc is a callback type for determining the return type of a function
// given its arguments.
//
// Any of the values passed to this function may be unknown, even if the
// parameters are not configured to accept unknowns.
//
// If any of the given values are *not* unknown, the TypeFunc may use the
// values for pre-validation and for choosing the return type. For example,
// a hypothetical JSON-unmarshalling function could return
// cty.DynamicPseudoType if the given JSON string is unknown, but return
// a concrete type based on the JSON structure if the JSON string is already
// known.
type TypeFunc func(args []cty.Value) (cty.Type, error)

// ImplFunc is a callback type for the main implementation of a function.
//
// "args" are the values for the arguments, and this slice will always be at
// least as long as the argument definition slice for the function.
//
// "retType" is the type returned from the Type callback, included as a
// convenience to avoid the need to re-compute the return type for generic
// functions whose return type is a function of the arguments.
type ImplFunc func(args []cty.Value, retType cty.Type) (cty.Value, error)

// StaticReturnType returns a TypeFunc that always returns the given type.
//
// This is provided as a convenience for defining a function whose return
// type does not depend on the argument types.
func StaticReturnType(ty cty.Type) TypeFunc {
	return func([]cty.Value) (cty.Type, error) {
		return ty, nil
	}
}

// ReturnType returns the return type of a function given a set of candidate
// argument types, or returns an error if the given types are unacceptable.
//
// If the caller already knows values for at least some of the arguments
// it can be better to call ReturnTypeForValues, since certain functions may
// determine their return types from their values and return DynamicVal if
// the values are unknown.
func (f Function) ReturnType(argTypes []cty.Type) (cty.Type, error) {
	vals := make([]cty.Value, len(argTypes))
	for i, ty := range argTypes {
		vals[i] = cty.UnknownVal(ty)
	}
	return f.ReturnTypeForValues(vals)
}

// ReturnTypeForValues is similar to ReturnType but can be used if the caller
// already knows the values of some or all of the arguments, in which case
// the function may be able to determine a more definite result if its
// return type depends on the argument *values*.
//
// For any arguments whose values are not known, pass an Unknown value of
// the appropriate type.
func (f Function) ReturnTypeForValues(args []cty.Value) (ty cty.Type, err error) {
	var posArgs []cty.Value
	var varArgs []cty.Value

	if f.spec.VarParam == nil {
		if len(args) != len(f.spec.Params) {
			return cty.Type{}, fmt.Errorf(
				"wrong number of arguments (%d required; %d given)",
				len(f.spec.Params), len(args),
			)
		}

		posArgs = args
		varArgs = nil
	} else {
		if len(args) < len(f.spec.Params) {
			return cty.Type{}, fmt.Errorf(
				"wrong number of arguments (at least %d required; %d given)",
				len(f.spec.Params), len(args),
			)
		}

		posArgs = args[0:len(f.spec.Params)]
		varArgs = args[len(f.spec.Params):]
	}

	for i, spec := range f.spec.Params {
		val := posArgs[i]

		if val.IsMarked() && !spec.AllowMarked {
			// During type checking we just unmark values and discard their
			// marks, under the assumption that during actual execution of
			// the function we'll do similarly and then re-apply the marks
			// afterwards. Note that this does mean that a function that
			// inspects values (rather than just types) in its Type
			// implementation can potentially fail to take into account marks,
			// unless it specifically opts in to seeing them.
			unmarked, _ := val.Unmark()
			newArgs := make([]cty.Value, len(args))
			copy(newArgs, args)
			newArgs[i] = unmarked
			args = newArgs
		}

		if val.IsNull() && !spec.AllowNull {
			return cty.Type{}, NewArgErrorf(i, "argument must not be null")
		}

		// AllowUnknown is ignored for type-checking, since we expect to be
		// able to type check with unknown values. We *do* still need to deal
		// with DynamicPseudoType here though, since the Type function might
		// not be ready to deal with that.

		if val.Type() == cty.DynamicPseudoType {
			if !spec.AllowDynamicType {
				return cty.DynamicPseudoType, nil
			}
		} else if errs := val.Type().TestConformance(spec.Type); errs != nil {
			// For now we'll just return the first error in the set, since
			// we don't have a good way to return the whole list here.
			// Would be good to do something better at some point...
			return cty.Type{}, NewArgError(i, errs[0])
		}
	}

	if varArgs != nil {
		spec := f.spec.VarParam
		for i, val := range varArgs {
			realI := i + len(posArgs)

			if val.IsMarked() && !spec.AllowMarked {
				// See the similar block in the loop above for what's going on here.
				unmarked, _ := val.Unmark()
				newArgs := make([]cty.Value, len(args))
				copy(newArgs, args)
				newArgs[realI] = unmarked
				args = newArgs
			}

			if val.IsNull() && !spec.AllowNull {
				return cty.Type{}, NewArgErrorf(realI, "argument must not be null")
			}

			if val.Type() == cty.DynamicPseudoType {
				if !spec.AllowDynamicType {
					return cty.DynamicPseudoType, nil
				}
			} else if errs := val.Type().TestConformance(spec.Type); errs != nil {
				// For now we'll just return the first error in the set, since
				// we don't have a good way to return the whole list here.
				// Would be good to do something better at some point...
				return cty.Type{}, NewArgError(i, errs[0])
			}
		}
	}

	// Intercept any panics from the function and return them as normal errors,
	// so a calling language runtime doesn't need to deal with panics.
	defer func() {
		if r := recover(); r != nil {
			ty = cty.NilType
			err = errorForPanic(r)
		}
	}()

	return f.spec.Type(args)
}

// Call actually calls the function with the given arguments, which must
// conform to the function's parameter specification or an error will be
// returned.
func (f Function) Call(args []cty.Value) (val cty.Value, err error) {
	expectedType, err := f.ReturnTypeForValues(args)
	if err != nil {
		return cty.NilVal, err
	}

	// Type checking already dealt with most situations relating to our
	// parameter specification, but we still need to deal with unknown
	// values and marked values.
	posArgs := args[:len(f.spec.Params)]
	varArgs := args[len(f.spec.Params):]
	var resultMarks []cty.ValueMarks

	for i, spec := range f.spec.Params {
		val := posArgs[i]

		if !val.IsKnown() && !spec.AllowUnknown {
			return cty.UnknownVal(expectedType), nil
		}

		if !spec.AllowMarked {
			unwrappedVal, marks := val.UnmarkDeep()
			if len(marks) > 0 {
				// In order to avoid additional overhead on applications that
				// are not using marked values, we copy the given args only
				// if we encounter a marked value we need to unmark. However,
				// as a consequence we end up doing redundant copying if multiple
				// marked values need to be unwrapped. That seems okay because
				// argument lists are generally small.
				newArgs := make([]cty.Value, len(args))
				copy(newArgs, args)
				newArgs[i] = unwrappedVal
				resultMarks = append(resultMarks, marks)
				args = newArgs
			}
		}
	}

	if f.spec.VarParam != nil {
		spec := f.spec.VarParam
		for i, val := range varArgs {
			if !val.IsKnown() && !spec.AllowUnknown {
				return cty.UnknownVal(expectedType), nil
			}
			if !spec.AllowMarked {
				unwrappedVal, marks := val.UnmarkDeep()
				if len(marks) > 0 {
					newArgs := make([]cty.Value, len(args))
					copy(newArgs, args)
					newArgs[len(posArgs)+i] = unwrappedVal
					resultMarks = append(resultMarks, marks)
					args = newArgs
				}
			}
		}
	}

	var retVal cty.Value
	{
		// Intercept any panics from the function and return them as normal errors,
		// so a calling language runtime doesn't need to deal with panics.
		defer func() {
			if r := recover(); r != nil {
				val = cty.NilVal
				err = errorForPanic(r)
			}
		}()

		retVal, err = f.spec.Impl(args, expectedType)
		if err != nil {
			return cty.NilVal, err
		}
		if len(resultMarks) > 0 {
			retVal = retVal.WithMarks(resultMarks...)
		}
	}

	// Returned value must conform to what the Type function expected, to
	// protect callers from having to deal with inconsistencies.
	if errs := retVal.Type().TestConformance(expectedType); errs != nil {
		panic(fmt.Errorf(
			"returned value %#v does not conform to expected return type %#v: %s",
			retVal, expectedType, errs[0],
		))
	}

	return retVal, nil
}

// ProxyFunc the type returned by the method Function.Proxy.
type ProxyFunc func(args ...cty.Value) (cty.Value, error)

// Proxy returns a function that can be called with cty.Value arguments
// to run the function. This is provided as a convenience for when using
// a function directly within Go code.
func (f Function) Proxy() ProxyFunc {
	return func(args ...cty.Value) (cty.Value, error) {
		return f.Call(args)
	}
}

// Params returns information about the function's fixed positional parameters.
// This does not include information about any variadic arguments accepted;
// for that, call VarParam.
func (f Function) Params() []Parameter {
	new := make([]Parameter, len(f.spec.Params))
	copy(new, f.spec.Params)
	return new
}

// VarParam returns information about the variadic arguments the function
// expects, or nil if the function is not variadic.
func (f Function) VarParam() *Parameter {
	if f.spec.VarParam == nil {
		return nil
	}

	ret := *f.spec.VarParam
	return &ret
}
