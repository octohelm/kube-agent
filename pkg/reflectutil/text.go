package reflectutil

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
)

var (
	ErrUnsupportedType = fmt.Errorf("unsupported type")
)

func MarshalText(v interface{}) ([]byte, error) {
	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(&v).Elem()
	}

	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, nil
	}

	if tm, ok := rv.Interface().(encoding.TextMarshaler); ok {
		return tm.MarshalText()
	}

	switch rv.Kind() {
	case reflect.Interface, reflect.Ptr:
		if rv.IsNil() {
			return nil, nil
		}
		return MarshalText(rv.Elem())
	case reflect.String:
		return []byte(rv.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(rv.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(rv.Uint(), 10)), nil
	case reflect.Bool:
		return []byte(strconv.FormatBool(rv.Bool())), nil
	case reflect.Float32, reflect.Float64:
		return []byte(strconv.FormatFloat(rv.Float(), 'f', -1, 64)), nil
	default:
		return nil, ErrUnsupportedType
	}
}

func UnmarshalText(v interface{}, data []byte) error {
	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}

	if rv.CanAddr() {
		if textUnmarshaler, ok := rv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if err := textUnmarshaler.UnmarshalText(data); err != nil {
				return err
			}
			return nil
		}
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			if rv.CanSet() {
				rv.Set(reflect.New(rv.Type().Elem()))
			}
		}
		return UnmarshalText(rv.Elem(), data)
	case reflect.String:
		rv.SetString(string(data))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intV, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(intV).Convert(rv.Type()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintV, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(uintV).Convert(rv.Type()))
	case reflect.Float32, reflect.Float64:
		floatV, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(floatV).Convert(rv.Type()))
	case reflect.Bool:
		boolV, err := strconv.ParseBool(string(data))
		if err != nil {
			return err
		}
		rv.SetBool(boolV)
	}
	return nil
}
