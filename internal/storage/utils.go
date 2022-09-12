package storage

import (
	"strconv"
)

// ToInt64 Конвертирование значения в int64
func ToInt64(value interface{}) (int64, error) {

	switch i := value.(type) {
	// int
	case int:
		return int64(i), nil
	case int8:
		return int64(i), nil
	case int16:
		return int64(i), nil
	case int32:
		return int64(i), nil
	case int64:
		return i, nil

	//	unsigned int
	case uint:
		return int64(i), nil
	case uint8:
		return int64(i), nil
	case uint16:
		return int64(i), nil
	case uint32:
		return int64(i), nil
	case uint64:
		return int64(i), nil

	case string:
		val, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			return 0, ErrInvalidValue
		}
		return val, nil

	default:
		return 0, ErrInvalidValue
	}
}

func ToFloat64(value interface{}) (float64, error) {

	switch i := value.(type) {
	case float32:
		return float64(i), nil
	case float64:
		return i, nil

	case int:
		return float64(i), nil
	case int8:
		return float64(i), nil
	case int16:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int64:
		return float64(i), nil

	case uint:
		return float64(i), nil
	case uint8:
		return float64(i), nil
	case uint16:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint64:
		return float64(i), nil

	case string:
		val, err := strconv.ParseFloat(i, 64)
		if err != nil {
			return 0, ErrInvalidValue
		}
		return val, nil

	default:
		return 0, ErrInvalidValue
	}
}
