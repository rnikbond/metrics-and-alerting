package storage

import (
	"reflect"
	"strconv"

	errst "metrics-and-alerting/pkg/errorsstorage"
)

func ToInt64(value interface{}) (int64, error) {
	reflVal := reflect.ValueOf(value)

	switch reflVal.Kind() {
	case
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		return reflVal.Int(), nil
	case
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:

		return int64(reflVal.Uint()), nil
	case reflect.String:
		val, err := strconv.ParseInt(reflVal.String(), 10, 64)
		if err != nil {
			return 0, errst.ErrorInvalidValue
		}
		return val, nil
	default:
		return 0, errst.ErrorInvalidValue
	}
}

func ToFloat64(value interface{}) (float64, error) {
	reflVal := reflect.ValueOf(value)

	switch reflVal.Kind() {
	case
		reflect.Float32,
		reflect.Float64:

		return reflVal.Float(), nil
	case
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		return float64(reflVal.Int()), nil
	case
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:

		return float64(reflVal.Uint()), nil
	case reflect.String:

		val, err := strconv.ParseFloat(reflVal.String(), 64)
		if err != nil {
			return 0, errst.ErrorInvalidValue
		}
		return val, nil
	default:
		return 0, errst.ErrorInvalidValue
	}
}
