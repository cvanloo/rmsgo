package rmsgo

import (
	"fmt"
)

// LDjson aliases instead of defines a new type to make unmarshalling easier.
type LDjson = map[string]any

func ldget[T any](ld LDjson, key string) (T, error) {
	var z T
	if v, ok := ld[key]; ok {
		if t, ok := v.(T); ok {
			return t, nil
		}
		return z, fmt.Errorf("%s: value `%v' of type %T cannot be cast to %T", key, v, v, z)
	}
	return z, fmt.Errorf("%s: no such entry in ldjson map", key)
}

// LDGet retrieves a value of type T from a nested ld+json map.
// It recursively follows the keys to reach the final value.
func LDGet[T any](ld LDjson, keys ...string) (t T, err error) {
	switch any(t).(type) {
	case float64:
	case string:
	case LDjson:
	case int:
		// because json.Unmarshal will parse any int as float64
		assert(false, "use float64 instead")
	default:
		assert(false, "invalid ldjson type")
	}

	assert(len(keys) > 0, "don't know what key to get")

	for _, key := range keys[:len(keys)-1] {
		ld, err = ldget[LDjson](ld, key)
		if err != nil {
			return
		}
	}
	return ldget[T](ld, keys[len(keys)-1])
}
