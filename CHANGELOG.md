# 1.8.0 (Unreleased)

* `cty`: When running on Go 1.16 or later, the `cty.String` type will now normalize incoming string values using the Unicode 13 normalization rules.
* `function/stdlib`: The various string functions which split strings into individual characters as part of their work will now use the Unicode 13 version of the text segmentation algorithm to do so.

# 1.7.2 (Unreleased)

* `cty`: The `Type.GoString` implementation for object types with optional attributes was previously producing incorrect results due to an implementation bug. ([#86](https://github.com/zclconf/go-cty/pull/86))

# 1.7.1 (Unreleased)

* `cty`: The `Value.Multiply` and `Value.Modulo` functions now correctly propagate the floating point precision of the arguments, which avoids generating incorrect results for large integer operands. ([#75](https://github.com/zclconf/go-cty/pull/75))
* `convert`: The `convert.MismatchMessage` function will now correctly identify mismatching attributes in objects, rather than misreporting attributes that are actually present and correct. ([#78](https://github.com/zclconf/go-cty/pull/78))
* `function/stdlib`: The `merge` function now returns an empty object if all of its arguments are `null`, rather than returning `null` as before. That's more consistent with its usual behavior of ignoring `null` arguments when there is at least one non-null argument. ([#82](https://github.com/zclconf/go-cty/pull/82))
* `function/stdlib`: The `coalescelist` function now ignores any arguments that are null, rather than panicking as before.. ([#81](https://github.com/zclconf/go-cty/pull/81))

# 1.7.0 (Unreleased)

* `cty`: `Value.UnmarkDeepWithPaths` and `Value.MarkWithPaths` are like `Value.UnmarkDeep` and `Value.Mark` but they retain path information for each marked value, so that marks can be re-applied later without all the loss of detail that results from `Value.UnmarkDeep` aggregating together all of the nested marks.
* `function`: Unless a parameter has `AllowMarks: true` explicitly set, the functions infrastructure will now guarantee that it never sees a marked value even if the mark is deep inside a data structure. Previously that guarantee was only shallow for the top-level value, similar to `AllowUnknown`, but because marks are a relatively new addition to `cty` and numerous existing functions are not written to deal with them this is the more conservative and robust default. ([#72](https://github.com/zclconf/go-cty/pull/72))
* `function/stdlib`: The `formatdate` function was not correctly handling literal sequences at the end of the format string. It will now handle those as intended. ([#69](https://github.com/zclconf/go-cty/pull/69))

# 1.6.1 (Unreleased)

* `cty`:: Fix a regression from 1.6.0 where `Value.RawEqual` no longer returned the correct result given a pair of sets containing partially-unknown values. ([#64](https://github.com/zclconf/go-cty/pull/64))

# 1.6.0 (Unreleased)

* Fixed various defects in the handling of sets containing unknown values. This will cause unknown values to now be returned in more situations, whereas before `cty` would often return incorrect results when working with sets containing unknown values. The list of defects fixed in this release includes:
    - `cty`: The length of a set containing unknown values, as defined by `Value.Length`, is itself unknown, reflecting the fact that unknown values may be placeholders for values that are equal to other values in the set, which would thus coalesce into a single value.
    - `cty:` Converting a set with unknown values to a list produces an unknown value, because type conversion can't predict which indices each element of the set should take (the unknown elements could appear anywhere in the sort order) or the length of the resulting list.
    - `function/stdlib`: the `LengthFunc` and `ToList` functions wrap the behaviors described in the previous two items and are therefore also fixed in the same way.
    - `function/stclib`: `FormatListFunc` depends on knowing the length of all of its sequence arguments (which includes support for sets), so it will return an unknown result if given a set with an unknown length.
    - `function/stdlib`: The various set operation functions were previously producing incorrect results if one of their given sets contained unknown values, because they didn't consider that unknown values on one set may be placeholders for values that are equal to elements of the other set. For example, `SetSubtractFunc` now produces a wholly-unknown result if either of its arguments contains an unknown element, because it can't predict whether that unknown element represents a value equal to an element in the other set.
    - `cty`: The `Value.Equal` function would previously incorrectly return a known `cty.False` if one of the given sets contained an unknown value. It will now return `cty.UnknownVal(cty.Bool)` in that case, reflecting that the result could be either `cty.True` or `cty.False` were the unknown values to be replaced with known values.
    - `cty`: The `Value.LengthInt` function was also returning incorrect results for sets containing unknown elements. However, given that it is commonly used in conjunction with `ElementIterator` to determine the capacity for a slice to append elements to, it is not fixed and is instead redefined to return the _maximum possible length_, which would result if all of the unknown values represent values that are not equal to any other set element. Applications that use `Value.LengthInt` to determine lengths to return to users who are working in the space of `cty` values should switch to using `Value.Length` instead and handle the possibility of the length being unknown, to avoid returning incorrect results for sets with unknown values.

    These are not classified as breaking changes because the previous behavior was defective per the design goals for unknown values. However, callers may notice their application behavior changes along with these fixes when upgrading. The new behaviors should all be more correct than the old; if you observe a change in behavior where there is now an _incorrect_ result for sets containing unknown values (that is, where `cty` claims it knows an answer that it should not actually know), please report that in a GitHub issue.

    We advise callers which work with sets that may potentially contain unknown values to review their own set-handling functions to check if they too might be handling sets with unknown values incorrectly, particularly if they work with sets using [integration methods rather than operation methods](./docs/types.md#common-operations-and-integration-methods) (for example, using `Value.ValueList` or `Value.ValueSet` to extract elements directly). It seems that incorrect handling of sets with unknown values has been a common hazard, particularly in codepaths that aim to treat lists and sets as being interchangable.
* `function/stdlib`: The `element` function will no longer panic if given a negative index. Instead, it will return a proper error. ([#62](https://github.com/zclconf/go-cty/pull/62))
* `convert`: **Experimental** support for annotating one or more attributes of an object type as "optional", which the `convert` package can then use to suppress the error that would normally be returned if the source type has no corresponding attribute, and can substitute a correctly-typed null value instead. This new behavior is subject to change even in minor release of `cty`, until it has been tested in experimental releases of downstream applications and potentially modified in response.

# 1.5.1 (Unreleased)

* `function/stdlib`: The `merge` function will no longer panic if all given maps are empty. ([#58](https://github.com/zclconf/go-cty/pull/58))
* `function/stdlib`: The various set-manipulation functions, like `setunion`, will no longer panic if given an unknown set value. ([#59](https://github.com/zclconf/go-cty/pull/59))

# 1.5.0 (March 17, 2025)

* `cty`: New `Value.HasWhollyKnownType` method, for testing whether a value's type could potentially change if any unknown values it was constructed from were to become known. ([#55](https://github.com/zclconf/go-cty/pull/55))
* `convert`: Fix incorrect panic when converting a tuple with a dynamic-typed null member into a list or set, due to overly-liberal type unification. ([#56](https://github.com/zclconf/go-cty/pull/56))

# 1.4.2

* `function/stdlib`: The `jsonencode` function will now correctly accept a null as its argument, and produce the JSON representation `"null"` rather than returning an error. ([#54](https://github.com/zclconf/go-cty/pull/54))
* `convert`: Don't panic when asked to convert a tuple of objects to a list type constraint containing a nested `cty.DynamicPseudoType`. ([#53](https://github.com/zclconf/go-cty/pull/53))

# 1.4.1 (March 5, 2025)

* `function/stdlib`: Fix various panics related to sets with unknown element types in the set-manipulation functions. ([#52](https://github.com/zclconf/go-cty/pull/52))
* `json`: Remove `json.UnmarshalDynamicWithImpliedType` function that was only available in hashicorp/go-cty 1.4.1 pseudo-versions. ([#6](https://github.com/hashicorp/go-cty/pull/6))

# 1.4.0 (April 7, 2020)

* `function/stdlib`: The string functions that partition strings into individual characters (grapheme clusters) now use the appropriate segmentation rules from Unicode 12.0.0, while previous versions used Unicode 9.0.0.
* `function/stdlib`: New functions `Replace` and `RegexReplace` for matching and replacing sequences of characters in a given string with another given string. ([#45](https://github.com/zclconf/go-cty/pull/45))
* `function/stdlib`: The function `Substr` will now produce a zero-length string when given a length of zero. Previously it was incorrectly returning the remainder of the string after the given offset. ([#48](https://github.com/zclconf/go-cty/pull/48))
* `function/stdlib`: The `Floor` and `Ceil` functions will now return an infinity if given an infinity, rather than returning the maximum/minimum integer value. ([#51](https://github.com/zclconf/go-cty/pull/51))
* `cty`: Convenience methods for constructing path index steps from normal Go int and string values. ([#50](https://github.com/zclconf/go-cty/pull/50))

# 1.3.1 (March 3, 2020)

* `convert`: Fix incorrect conversion rules for maps of maps that were leading to panics. This will now succeed in some more cases that ought to have been valid, and produce a proper error if there is no valid outcome. ([#47](https://github.com/zclconf/go-cty/pull/47))
* `function/stdlib`: Fix an implementation error in the `Contains` function that was introduced in 1.3.0, so it will now produce a correct result rather than failing with a confusing error message. ([#46](https://github.com/zclconf/go-cty/pull/46))

# 1.3.0 (February 19, 2020)

* `convert`: There are now conversions from map types to object types, as long as the given map type's element type is convertible to all of the object type's attribute types. ([#42](https://github.com/zclconf/go-cty/pull/42))
* `function/stdlib`: HashiCorp has contributed a number of additional functions to the standard library that were originally implemented directly inside their Terraform codebase: ([#37](https://github.com/zclconf/go-cty/pull/37))
  * `Element`: take an element from a list or tuple by index, using modulo wrap-around.
  * `CoalesceList`: return the first non-empty list argument.
  * `Compact`: take a list of strings and return a new list of strings with all empty strings removed.
  * `Contains`: returns true if a given value appears as an element in a list, tuple, or set.
  * `Distinct`: filters duplicate elements from a list while retaining the order of remaining items.
  * `ChunkList`: turn a list into a list-of-lists where each top-level list is a "chunk" of a particular size of elements from the input.
  * `Flatten`: given a sequence that might contain other sequences, eliminate any intermediate sequences to produce a flat sequence.
  * `Keys`: return a list of keys from a map or object value in lexical order.
  * `Values`: return a list of values from a map in the same order as `Keys`.
  * `Lookup`: conditional lookup of an element from a map if it's present, or a fallback value if not. (This one differs from its Terraform equivalent in that the default value argument is _required_.)
  * `Merge`: given one or more maps or objects, merge them together into a single collection.
  * `ReverseList`: given a list, return a new list with the same items in the opposite order.
  * `SetProduct`: compute the cartesian product of one or more sets.
  * `Slice`: extract a consecutive sub-list from a list.
  * `Zipmap`: given a pair of lists of the same length, interpret the first as keys and the second as corresponding values to produce a map.
  * A factory `MakeToFunc` to build functions that each convert to a particular type constraint.
  * `TimeAdd`: add a duration to a timestamp to produce a new timestamp.
  * `Ceil` and `Floor`: round a fractional value to the nearest integer, away from or towards zero respectively.
  * `Log`: computes a logarithm in a given base.
  * `Pow`: implements exponentiation.
  * `ParseInt`: parses a string containing digits in a particular base to produce a whole number value.
  * `Join`: concatenates the elements of a list of strings with a given separator to produce a string.
  * `Split`: partitions a string by a given separator, returning a list of strings.
  * `Sort`: sorts a list of strings into lexical order.
  * `Chomp`: removes one or more newline characters from the end of a given string, producing a new string.
  * `Indent`: prepends a number of spaces to all lines except the first in a given string, producing a new string.
  * `Title`: converts a string to "title case".
  * `TrimSpace`: trims spaces from the start and end of a given string.
  * `Trim`: generalization of `TrimSpace` that allows user-specified trimming characters.
  * `TrimPrefix`: like `Trim` but only at the start of the string.
  * `TrimSuffix`: like `Trim` but only at the end of the string.

# 1.2.1 (January 10, 2020)

* `cty`: Fixed an infinite recursion bug when working with sets containing nested data structures. ([#35](https://github.com/zclconf/go-cty/pull/35))

# 1.2.0 (December 14, 2019)

* `cty`: Applications can now implement a general subset of the `cty` operations when creating a capsule type. For more information, see [Capsule Type Operation Definitions](./docs/capsule-type-operations.md).
* `cty`: Values now support a new mechanism called [Value Marks](./docs/marks.md) which can be used to transit additional metadata through expressions by marking the input values and then observing which marks propagated to the result value. This could be used, for example, to detect whether a value was derived from a particular other value in case that is useful for giving extra feedback in an error message.

# 1.1.1 (November 26, 2019)

* `cty`: Fixed a panic situation when trying to round-trip `cty.Number` values
  through `encoding/gob`. ([#32](https://github.com/zclconf/go-cty/pull/32))
* `convert`: Invalid string conversions to bool that use incorrect case will now give more actionable feedback. ([#29](https://github.com/zclconf/go-cty/pull/29))
* `function/stdlib`: The `formatlist` function will no longer panic if given
  an unknown tuple as one of its arguments.

# 1.1.0 (July 25, 2019)

* New method `Path.Equals` for robustly comparing `cty.Path` values. Previously
  callers might've used `reflect.DeepEqual` or similar, but that is not
  correct when a path contains a `cty.Number` index because `reflect.DeepEqual`
  does not correctly represent equality for number values.
  ([#25](https://github.com/zclconf/go-cty/pull/25))

# 1.0.0 (June 6, 2019)

Initial stable release.
