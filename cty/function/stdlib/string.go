package stdlib

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/apparentlymart/go-textseg/v13/textseg"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/function"
	"github.com/hashicorp/go-cty/cty/gocty"
)

var UpperFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := strings.ToUpper(in)
		return cty.StringVal(out), nil
	},
})

var LowerFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := strings.ToLower(in)
		return cty.StringVal(out), nil
	},
})

var ReverseFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := []byte(args[0].AsString())
		out := make([]byte, len(in))
		pos := len(out)

		inB := []byte(in)
		for i := 0; i < len(in); {
			d, _, _ := textseg.ScanGraphemeClusters(inB[i:], true)
			cluster := in[i : i+d]
			pos -= len(cluster)
			copy(out[pos:], cluster)
			i += d
		}

		return cty.StringVal(string(out)), nil
	},
})

var StrlenFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		l := 0

		inB := []byte(in)
		for i := 0; i < len(in); {
			d, _, _ := textseg.ScanGraphemeClusters(inB[i:], true)
			l++
			i += d
		}

		return cty.NumberIntVal(int64(l)), nil
	},
})

var SubstrFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "str",
			Type:             cty.String,
			AllowDynamicType: true,
		},
		{
			Name:             "offset",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
		{
			Name:             "length",
			Type:             cty.Number,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := []byte(args[0].AsString())
		var offset, length int

		var err error
		err = gocty.FromCtyValue(args[1], &offset)
		if err != nil {
			return cty.NilVal, err
		}
		err = gocty.FromCtyValue(args[2], &length)
		if err != nil {
			return cty.NilVal, err
		}

		if offset < 0 {
			totalLenNum, err := Strlen(args[0])
			if err != nil {
				// should never happen
				panic("Stdlen returned an error")
			}

			var totalLen int
			err = gocty.FromCtyValue(totalLenNum, &totalLen)
			if err != nil {
				// should never happen
				panic("Stdlen returned a non-int number")
			}

			offset += totalLen
		} else if length == 0 {
			// Short circuit here, after error checks, because if a
			// string of length 0 has been requested it will always
			// be the empty string
			return cty.StringVal(""), nil
		}

		sub := in
		pos := 0
		var i int

		// First we'll seek forward to our offset
		if offset > 0 {
			for i = 0; i < len(sub); {
				d, _, _ := textseg.ScanGraphemeClusters(sub[i:], true)
				i += d
				pos++
				if pos == offset {
					break
				}
				if i >= len(in) {
					return cty.StringVal(""), nil
				}
			}

			sub = sub[i:]
		}

		if length < 0 {
			// Taking the remainder of the string is a fast path since
			// we can just return the rest of the buffer verbatim.
			return cty.StringVal(string(sub)), nil
		}

		// Otherwise we need to start seeking forward again until we
		// reach the length we want.
		pos = 0
		for i = 0; i < len(sub); {
			d, _, _ := textseg.ScanGraphemeClusters(sub[i:], true)
			i += d
			pos++
			if pos == length {
				break
			}
		}

		sub = sub[:i]

		return cty.StringVal(string(sub)), nil
	},
})

var JoinFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "separator",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name: "lists",
		Type: cty.List(cty.String),
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		sep := args[0].AsString()
		listVals := args[1:]
		if len(listVals) < 1 {
			return cty.UnknownVal(cty.String), fmt.Errorf("at least one list is required")
		}

		l := 0
		for _, list := range listVals {
			if !list.IsWhollyKnown() {
				return cty.UnknownVal(cty.String), nil
			}
			l += list.LengthInt()
		}

		items := make([]string, 0, l)
		for ai, list := range listVals {
			ei := 0
			for it := list.ElementIterator(); it.Next(); {
				_, val := it.Element()
				if val.IsNull() {
					if len(listVals) > 1 {
						return cty.UnknownVal(cty.String), function.NewArgErrorf(ai+1, "element %d of list %d is null; cannot concatenate null values", ei, ai+1)
					}
					return cty.UnknownVal(cty.String), function.NewArgErrorf(ai+1, "element %d is null; cannot concatenate null values", ei)
				}
				items = append(items, val.AsString())
				ei++
			}
		}

		return cty.StringVal(strings.Join(items, sep)), nil
	},
})

var SortFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "list",
			Type: cty.List(cty.String),
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		listVal := args[0]

		if !listVal.IsWhollyKnown() {
			// If some of the element values aren't known yet then we
			// can't yet predict the order of the result.
			return cty.UnknownVal(retType), nil
		}
		if listVal.LengthInt() == 0 { // Easy path
			return listVal, nil
		}

		list := make([]string, 0, listVal.LengthInt())
		for it := listVal.ElementIterator(); it.Next(); {
			iv, v := it.Element()
			if v.IsNull() {
				return cty.UnknownVal(retType), fmt.Errorf("given list element %s is null; a null string cannot be sorted", iv.AsBigFloat().String())
			}
			list = append(list, v.AsString())
		}

		sort.Strings(list)
		retVals := make([]cty.Value, len(list))
		for i, s := range list {
			retVals[i] = cty.StringVal(s)
		}
		return cty.ListVal(retVals), nil
	},
})

var SplitFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "separator",
			Type: cty.String,
		},
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		sep := args[0].AsString()
		str := args[1].AsString()
		elems := strings.Split(str, sep)
		elemVals := make([]cty.Value, len(elems))
		for i, s := range elems {
			elemVals[i] = cty.StringVal(s)
		}
		if len(elemVals) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}
		return cty.ListVal(elemVals), nil
	},
})

// ChompFunc is a function that removes newline characters at the end of a
// string.
var ChompFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		newlines := regexp.MustCompile(`(?:\r\n?|\n)*\z`)
		return cty.StringVal(newlines.ReplaceAllString(args[0].AsString(), "")), nil
	},
})

// IndentFunc is a function that adds a given number of spaces to the
// beginnings of all but the first line in a given multi-line string.
var IndentFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "spaces",
			Type: cty.Number,
		},
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var spaces int
		if err := gocty.FromCtyValue(args[0], &spaces); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		data := args[1].AsString()
		pad := strings.Repeat(" ", spaces)
		return cty.StringVal(strings.Replace(data, "\n", "\n"+pad, -1)), nil
	},
})

// TitleFunc is a function that converts the first letter of each word in the
// given string to uppercase.
var TitleFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.StringVal(strings.Title(args[0].AsString())), nil
	},
})

// TrimSpaceFunc is a function that removes any space characters from the start
// and end of the given string.
var TrimSpaceFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return cty.StringVal(strings.TrimSpace(args[0].AsString())), nil
	},
})

// TrimFunc is a function that removes the specified characters from the start
// and end of the given string.
var TrimFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
		{
			Name: "cutset",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		cutset := args[1].AsString()
		return cty.StringVal(strings.Trim(str, cutset)), nil
	},
})

// TrimPrefixFunc is a function that removes the specified characters from the
// start the given string.
var TrimPrefixFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
		{
			Name: "prefix",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		prefix := args[1].AsString()
		return cty.StringVal(strings.TrimPrefix(str, prefix)), nil
	},
})

// TrimSuffixFunc is a function that removes the specified characters from the
// end of the given string.
var TrimSuffixFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
		{
			Name: "suffix",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		cutset := args[1].AsString()
		return cty.StringVal(strings.TrimSuffix(str, cutset)), nil
	},
})

// Upper is a Function that converts a given string to uppercase.
func Upper(str cty.Value) (cty.Value, error) {
	return UpperFunc.Call([]cty.Value{str})
}

// Lower is a Function that converts a given string to lowercase.
func Lower(str cty.Value) (cty.Value, error) {
	return LowerFunc.Call([]cty.Value{str})
}

// Reverse is a Function that reverses the order of the characters in the
// given string.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
func Reverse(str cty.Value) (cty.Value, error) {
	return ReverseFunc.Call([]cty.Value{str})
}

// Strlen is a Function that returns the length of the given string in
// characters.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
func Strlen(str cty.Value) (cty.Value, error) {
	return StrlenFunc.Call([]cty.Value{str})
}

// Substr is a Function that extracts a sequence of characters from another
// string and creates a new string.
//
// As usual, "character" for the sake of this function is a grapheme cluster,
// so combining diacritics (for example) will be considered together as a
// single character.
//
// The "offset" index may be negative, in which case it is relative to the
// end of the given string.
//
// The "length" may be -1, in which case the remainder of the string after
// the given offset will be returned.
func Substr(str cty.Value, offset cty.Value, length cty.Value) (cty.Value, error) {
	return SubstrFunc.Call([]cty.Value{str, offset, length})
}

// Join concatenates together the string elements of one or more lists with a
// given separator.
func Join(sep cty.Value, lists ...cty.Value) (cty.Value, error) {
	args := make([]cty.Value, len(lists)+1)
	args[0] = sep
	copy(args[1:], lists)
	return JoinFunc.Call(args)
}

// Sort re-orders the elements of a given list of strings so that they are
// in ascending lexicographical order.
func Sort(list cty.Value) (cty.Value, error) {
	return SortFunc.Call([]cty.Value{list})
}

// Split divides a given string by a given separator, returning a list of
// strings containing the characters between the separator sequences.
func Split(sep, str cty.Value) (cty.Value, error) {
	return SplitFunc.Call([]cty.Value{sep, str})
}

// Chomp removes newline characters at the end of a string.
func Chomp(str cty.Value) (cty.Value, error) {
	return ChompFunc.Call([]cty.Value{str})
}

// Indent adds a given number of spaces to the beginnings of all but the first
// line in a given multi-line string.
func Indent(spaces, str cty.Value) (cty.Value, error) {
	return IndentFunc.Call([]cty.Value{spaces, str})
}

// Title converts the first letter of each word in the given string to uppercase.
func Title(str cty.Value) (cty.Value, error) {
	return TitleFunc.Call([]cty.Value{str})
}

// TrimSpace removes any space characters from the start and end of the given string.
func TrimSpace(str cty.Value) (cty.Value, error) {
	return TrimSpaceFunc.Call([]cty.Value{str})
}

// Trim removes the specified characters from the start and end of the given string.
func Trim(str, cutset cty.Value) (cty.Value, error) {
	return TrimFunc.Call([]cty.Value{str, cutset})
}

// TrimPrefix removes the specified prefix from the start of the given string.
func TrimPrefix(str, prefix cty.Value) (cty.Value, error) {
	return TrimPrefixFunc.Call([]cty.Value{str, prefix})
}

// TrimSuffix removes the specified suffix from the end of the given string.
func TrimSuffix(str, suffix cty.Value) (cty.Value, error) {
	return TrimSuffixFunc.Call([]cty.Value{str, suffix})
}
