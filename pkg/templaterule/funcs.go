package templaterule

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gopkg.in/yaml.v2"
)

// Funcs that are available to all Rules.
//
//	constraint
//	  Returns the korrel8r.Constraint in force when applying a rule. May be nil.
//	has
//	  Evaluates its arguments for errors. Useful for asserting that fields exist in the context value.
//	assert
//	  Exits with an error if its argument is false.
//	json
//	  Marshals its argument as JSON and returns the string.
//	yaml
//	  Marshals its argument as YAML and returns the string.
//	classname
//	  Returns the domain qualified name of a korrel8r.Class argument.
//	urlencode
//	  Returns the URL query encoding of a map argument.
//	  Map values are stringified as for fmt.Printf("%v", ...)
//	selector
//	  Takes a map arguments and returns a selector string of the form: "k1=value1,k2=value2 ..."
//	kvmap
//	  Returns a map formed from (key, value, key2, value2...) arguments.
//	  Useful for passing multiple parameters to a template execution.
var Funcs map[string]any

func init() {
	Funcs = map[string]any{
		"constraint": func() *korrel8r.Constraint { return nil },
		"assert":     doAssert, // Assert a condition in a template
		"json":       toJSON,
		"yaml":       toYAML,
		"classname":  korrel8r.ClassName,
		"urlencode":  urlencode,
		"selector":   selector,
		"kvmap":      kvMap,
	}
}

var errFailed = errors.New("failed")

// assert a condition in a template - int return is a dummy value required for template functions.
func doAssert(ok bool, msg ...any) (int, error) {
	if !ok {
		if len(msg) > 0 {
			if s, ok := msg[0].(string); ok {
				return 0, fmt.Errorf(s, msg[1:]...)
			}
		}
		return 0, errFailed
	}
	return 0, nil
}

func toJSON(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func toYAML(v any) (string, error) { b, err := yaml.Marshal(v); return string(b), err }

func urlencode(m any) string {
	v := reflect.ValueOf(m)
	if !v.IsValid() {
		return ""
	}
	p := url.Values{}
	i := v.MapRange()
	for i.Next() {
		p.Add(fmt.Sprintf("%v", i.Key()), fmt.Sprintf("%v", i.Value()))
	}
	return p.Encode()
}

func selector(m any) string {
	v := reflect.ValueOf(m)
	if !v.IsValid() {
		return ""
	}
	b := &strings.Builder{}
	i := v.MapRange()
	keys := make([]string, 0, v.Len())
	for i.Next() {
		keys = append(keys, i.Key().String())
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(b, "%v=%v", k, v.MapIndex(reflect.ValueOf(k)))
	}
	return b.String()
}

func kvMap(keyValue ...any) map[string]any {
	if len(keyValue) == 0 {
		return nil
	}
	m := map[string]any{}
	for i := 0; i < len(keyValue); i += 2 { // Will panic out of range on odd number
		m[keyValue[i].(string)] = keyValue[i+1]
	}
	return m
}