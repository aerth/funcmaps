package funcmaps

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"unicode"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
)

// most of this file copied from https://github.com/pthethanh/template/blob/master/funcs.go

var (
	errorType        = reflect.TypeOf((*error)(nil)).Elem()
	fmtStringerType  = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()

	zero       reflect.Value
	missingVal = reflect.ValueOf(missingValType{})
)

type (
	missingValType struct{}
	kind           int
)

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison     = errors.New("incompatible types for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
)

// AddFuncs adds to values the functions in funcs.
// It will panic if the func is not a good func or name is not a good name.
func AddFuncs(out, in map[string]interface{}) {
	for name, fn := range in {
		if !goodName(name) {
			panic(fmt.Sprintf("%s is not a good name", name))
		}
		if !goodFunc(reflect.TypeOf(fn)) {
			panic(fmt.Sprintf("%s is not a good func", fn))
		}
		out[name] = fn
	}
}

// goodFunc reports whether the function or method has the right result signature.
func goodFunc(typ reflect.Type) bool {
	// We allow functions with 1 result or 2 results where the second is an error.
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}
	return false
}

// goodName reports whether the function name is a valid identifier.
func goodName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_':
		case i == 0 && !unicode.IsLetter(r):
			return false
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			return false
		}
	}
	return true
}

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}
	return invalidKind, errBadComparisonType
}

// eq evaluates the comparison a == b || a == c || ...
func eq(arg1 reflect.Value, arg2 ...reflect.Value) (bool, error) {
	v1 := indirectInterface(arg1)
	if v1 != zero {
		if t1 := v1.Type(); !t1.Comparable() {
			return false, fmt.Errorf("uncomparable type %s: %v", t1, v1)
		}
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	k1, _ := basicKind(v1)
	for _, arg := range arg2 {
		v2 := indirectInterface(arg)
		k2, _ := basicKind(v2)
		truth := false
		if k1 != k2 {
			// Special case: Can compare integer values regardless of type's sign.
			switch {
			case k1 == intKind && k2 == uintKind:
				truth = v1.Int() >= 0 && uint64(v1.Int()) == v2.Uint()
			case k1 == uintKind && k2 == intKind:
				truth = v2.Int() >= 0 && v1.Uint() == uint64(v2.Int())
			default:
				return false, errBadComparison
			}
		} else {
			switch k1 {
			case boolKind:
				truth = v1.Bool() == v2.Bool()
			case complexKind:
				truth = v1.Complex() == v2.Complex()
			case floatKind:
				truth = v1.Float() == v2.Float()
			case intKind:
				truth = v1.Int() == v2.Int()
			case stringKind:
				truth = v1.String() == v2.String()
			case uintKind:
				truth = v1.Uint() == v2.Uint()
			default:
				if v2 == zero {
					truth = v1 == v2
				} else {
					if t2 := v2.Type(); !t2.Comparable() {
						return false, fmt.Errorf("uncomparable type %s: %v", t2, v2)
					}
					truth = v1.Interface() == v2.Interface()
				}
			}
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// indirect returns the item at the end of indirection, and a bool to indicate
// if it's nil. If the returned bool is true, the returned value's kind will be
// either a pointer or interface.
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}

// printableValue returns the, possibly indirected, interface value inside v that
// is best for a call to formatted printer.
func printableValue(v reflect.Value) interface{} {
	v = indirectInterface(v)
	if v.Kind() == reflect.Ptr {
		v, _ = indirect(v) // fmt.Fprint handles nil.
	}
	if !v.IsValid() {
		return ""
	}

	if !v.Type().Implements(errorType) && !v.Type().Implements(fmtStringerType) {
		if v.CanAddr() && (reflect.PtrTo(v.Type()).Implements(errorType) || reflect.PtrTo(v.Type()).Implements(fmtStringerType)) {
			v = v.Addr()
		} else {
			switch v.Kind() {
			case reflect.Chan, reflect.Func:
				return nil
			}
		}
	}
	return v.Interface()
}

// EqualAny return true if v equal to one of the values.
func EqualAny(v interface{}, values ...interface{}) bool {
	for _, val := range values {
		if ok, err := eq(reflect.ValueOf(v), reflect.ValueOf(val)); ok && err == nil {
			return true
		}
	}
	return false
}

// Repeat repeats the string representation of value n times.
func Repeat(n int, v interface{}) string {
	rs := &strings.Builder{}
	for i := 0; i < n; i++ {
		rs.WriteString(fmt.Sprintf("%v", printableValue(reflect.ValueOf(v))))
	}
	return rs.String()
}

// Join join the string representation of the values together.
// String will be joined as whole.
// Map, slice, array will be joined using its value, one by one.
func Join2(sep string, values ...interface{}) string {
	rs := make([]string, 0)
	for _, val := range values {
		v, isNil := indirect(reflect.ValueOf(val))
		if isNil {
			return ""
		}
		switch v.Kind() {
		case reflect.String:
			rs = append(rs, v.String())
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				rs = append(rs, fmt.Sprintf("%v", printableValue(v.Index(i))))
			}
		case reflect.Map:
			r := v.MapRange()
			for r.Next() {
				rs = append(rs, fmt.Sprintf("%v", printableValue(r.Value())))
			}
		default:
			rs = append(rs, fmt.Sprintf("%v", printableValue(v)))
		}
	}
	return strings.Join(rs, sep)
}

// Has check whether all the values exist in the collection.
// The collection must be a slice, array, string or a map.
func Has(collection reflect.Value, values ...reflect.Value) bool {
	for _, val := range values {
		if ok := has(collection, val); !ok {
			return false
		}
	}
	return true
}

// HasAny check whether one of the value exist in the collection.
// The collection must be a slice, array, string or a map.
func HasAny(collection reflect.Value, values ...reflect.Value) bool {
	for _, val := range values {
		if ok := has(collection, val); ok {
			return true
		}
	}
	return false
}

func has(collection reflect.Value, val reflect.Value) bool {
	v, isNil := indirect(collection)
	if isNil {
		return false
	}
	val, isNil = indirect(val)
	switch v.Kind() {
	case reflect.String:
		// accept all kinds of val.
		return strings.Contains(v.String(), fmt.Sprintf("%v", val))
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			iv, vIsNil := indirect(v.Index(i))
			// accept compare nil, nil
			if isNil && vIsNil || (!val.IsValid() && !iv.IsValid()) {
				return true
			}
			if ok, _ := eq(val, iv); ok {
				return true
			}
		}
	case reflect.Map:
		r := v.MapRange()
		for r.Next() {
			iv, vIsNil := indirect(r.Value())
			// accept compare nil, nil
			if isNil && vIsNil || (!val.IsValid() && !iv.IsValid()) {
				return true
			}
			if ok, _ := eq(iv, val); ok {
				return true
			}
		}
	default:
		return false
	}
	return false
}

// YesNo returns the first value if the last value has meaningful value/IsTrue, otherwise returns the second value.
func YesNo(v interface{}, y interface{}, n interface{}) interface{} {
	if IsTrue(v) {
		return y
	}
	return n
}

// Coalesce return first meaningful value (IsTrue).
func Coalesce(v ...interface{}) interface{} {
	for _, val := range v {
		if IsTrue(val) {
			return val
		}
	}
	return nil
}

// Default return default value if the given value is not meaningful (not IsTrue).
func IsDefault(df interface{}, v interface{}) interface{} {
	if IsEmpty(v) {
		return df
	}
	return v
}

// IsTrue reports whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value. This is the definition of
// truth used by if and other such actions.
func IsTrue(v interface{}) bool {
	if truth, ok := template.IsTrue(v); truth && ok {
		return ok
	}
	return false
}

// IsEmpty report whether the value not holding meaningful value.
// Opposite with IsTrue.
func IsEmpty(v interface{}) bool {
	return !IsTrue(v)
}

// UUID return a UUID.
func UUID() string {
	return uuid.New().String()
}

// FileSizeFormat return human readable string of file size.
func FileSizeFormat(value interface{}) string {
	var size float64

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		size = float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		size = float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		size = v.Float()
	default:
		return ""
	}

	var KB float64 = 1 << 10
	var MB float64 = 1 << 20
	var GB float64 = 1 << 30
	var TB float64 = 1 << 40
	var PB float64 = 1 << 50

	filesizeFormat := func(filesize float64, suffix string) string {
		return strings.Replace(fmt.Sprintf("%.1f %s", filesize, suffix), ".0", "", -1)
	}

	var result string
	if size < KB {
		result = filesizeFormat(size, "bytes")
	} else if size < MB {
		result = filesizeFormat(size/KB, "KB")
	} else if size < GB {
		result = filesizeFormat(size/MB, "MB")
	} else if size < TB {
		result = filesizeFormat(size/GB, "GB")
	} else if size < PB {
		result = filesizeFormat(size/TB, "TB")
	} else {
		result = filesizeFormat(size/PB, "PB")
	}

	return result
}

// Map return a map of string -> interface from provided key/value pairs.
func Map(v ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	lv := len(v)
	for i := 0; i < lv; i += 2 {
		key := fmt.Sprintf("%v", v[i])
		if i+1 >= lv {
			m[key] = ""
			continue
		}
		m[key] = v[i+1]
	}
	return m
}

var (
	textPolicy     = bluemonday.StripTagsPolicy()
	htmlPolicy     = bluemonday.UGCPolicy()
	initHTMLPolicy bool
)

// ASCIISpace ...
var ASCIISpace = rune(` `[0])

// StripTags allows everything, allows tabs, newlines, ASCII space but
// non-conforming whitespace, control chars, and also prevents shouting
func StripTags(s string) string {
	return textPolicy.Sanitize(stripChars(s, true, true, true, true))
}

// StripTagsSentence strips all HTML tags, allows ASCII space
func StripTagsSentence(s string) string {
	return textPolicy.Sanitize(stripChars(s, true, true, false, true))
}

// SanitiseHTML sanitizes HTML
// Leaving a safe set of HTML intact that is not going to pose an XSS risk
func Sanitize(s string) string {
	if !initHTMLPolicy {
		htmlPolicy.RequireNoFollowOnLinks(false)
		htmlPolicy.RequireNoFollowOnFullyQualifiedLinks(true)
		htmlPolicy.AddTargetBlankToFullyQualifiedLinks(true)
		initHTMLPolicy = true
	}

	return htmlPolicy.Sanitize(s)
}

// stripChars will remove unicode characters according to the instructions given
// to it. This is strictly speaking a whitelist, it's less "strip", more "allow".
//
// With SanitiseText this should run before bluemonday (HTML sanitiser)
//
// With SanitiseHTML this should run before blackfriday (Markdown generator)
func stripChars(
	in string,
	allowPrint bool,
	allowASCIISpace bool,
	allowFormattingSpace bool,
	preventShouting bool,
) string {
	tmp := []rune{}

	isShouting := true

	for _, runeValue := range in {
		var ret bool
		// IsPrint covers anything actually printable plus ASCII space
		if allowPrint && unicode.IsPrint(runeValue) {
			if !allowASCIISpace && unicode.IsSpace(runeValue) {
				continue
			}

			if isShouting && preventShouting && unicode.IsLower(runeValue) {
				isShouting = false
			}

			ret = true
		}

		// Only ASCII space
		if allowASCIISpace && runeValue == ASCIISpace {
			ret = true
		}

		// IsSpace covers tabs, newlines, ASCII space,etc
		if allowFormattingSpace && unicode.IsSpace(runeValue) {
			ret = true
		}

		if ret {
			tmp = append(tmp, runeValue)
		}
	}

	if isShouting && preventShouting {
		tmp2 := []rune{}
		for _, runeValue := range string(tmp) {
			tmp2 = append(tmp2, unicode.ToLower(runeValue))
		}
		tmp = tmp2
	}

	return string(tmp)
}
