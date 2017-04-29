package convert

import (
	"github.com/apparentlymart/go-cty/cty"
)

// compareTypes is the implementation of the public CompareTypes function,
// defined in public.go.
func compareTypes(a cty.Type, b cty.Type) int {

	if a == cty.DynamicPseudoType || b == cty.DynamicPseudoType {
		if a != cty.DynamicPseudoType {
			return 1
		}
		if b != cty.DynamicPseudoType {
			return -1
		}
		return 0
	}

	if a.IsPrimitiveType() && b.IsPrimitiveType() {
		// String is a supertype of all primitive types, because we can
		// represent all primitive values as specially-formatted strings.
		if a == cty.String || b == cty.String {
			if a != cty.String {
				return 1
			}
			if b != cty.String {
				return -1
			}
			return 0
		}
	}

	if a.IsListType() && b.IsListType() {
		return compareTypes(a.ElementType(), b.ElementType())
	}
	if a.IsSetType() && b.IsSetType() {
		return compareTypes(a.ElementType(), b.ElementType())
	}
	if a.IsMapType() && b.IsMapType() {
		return compareTypes(a.ElementType(), b.ElementType())
	}

	// TODO: list to tuple, map to object, set to list

	// For object and tuple types, comparing two types doesn't really tell
	// the whole story because it may be possible to construct a new type C
	// that is the supertype of both A and B by unifying each attribute/element
	// separately. That possibility is handled by Unify as a follow-up if
	// type sorting is insufficient to produce a valid result.
	//
	// Here we will take care of the simple possibilities where no new type
	// is needed.
	if a.IsObjectType() && b.IsObjectType() {
		atysA := a.AttributeTypes()
		atysB := b.AttributeTypes()

		if len(atysA) != len(atysB) {
			return 0
		}

		hasASuper := false
		hasBSuper := false
		for k := range atysA {
			if _, has := atysB[k]; !has {
				return 0
			}

			cmp := compareTypes(atysA[k], atysB[k])
			if cmp < 0 {
				hasASuper = true
			} else if cmp > 0 {
				hasBSuper = true
			}
		}

		switch {
		case hasASuper && hasBSuper:
			return 0
		case hasASuper:
			return -1
		case hasBSuper:
			return 1
		default:
			return 0
		}
	}
	if a.IsTupleType() && b.IsTupleType() {
		etysA := a.TupleElementTypes()
		etysB := b.TupleElementTypes()

		if len(etysA) != len(etysB) {
			return 0
		}

		hasASuper := false
		hasBSuper := false
		for i := range etysA {
			cmp := compareTypes(etysA[i], etysB[i])
			if cmp < 0 {
				hasASuper = true
			} else if cmp > 0 {
				hasBSuper = true
			}
		}

		switch {
		case hasASuper && hasBSuper:
			return 0
		case hasASuper:
			return -1
		case hasBSuper:
			return 1
		default:
			return 0
		}
	}

	return 0
}
