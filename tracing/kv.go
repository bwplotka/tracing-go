package tracing

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/attribute"
)

// borrowed from https://github.com/go-logfmt/logfmt/blob/main/encode.go#L75
func kvToAttr(keyvals ...interface{}) []attribute.KeyValue {
	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, nil)
	}

	j := 0
	ret := make([]attribute.KeyValue, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		k, ok := anyToString(keyvals[i])
		if !ok {
			k = "<unsupported key type>"
		}

		v, ok := anyToString(keyvals[i+1])
		if !ok {
			v = "<unsupported value type>"
		}

		ret[j] = attribute.String(k, v)
		j++
	}
	return ret
}

func anyToString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "nil", true
	case string:
		return v, true
	case []byte:
		return string(v), true
	case error:
		return v.Error(), true
	case fmt.Stringer:
		return v.String(), true
	default:
		rvalue := reflect.ValueOf(value)
		switch rvalue.Kind() {
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice, reflect.Struct:
			return "", false
		case reflect.Ptr:
			if rvalue.IsNil() {
				return "nil", true
			}
			return anyToString(rvalue.Elem().Interface())
		}
		return fmt.Sprint(v), true
	}
}
