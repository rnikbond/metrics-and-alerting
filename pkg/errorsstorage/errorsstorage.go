package errorsstorage

import (
	"errors"
	"net/http"
)

var (
	ErrorNotFound         = errors.New("metric not found")
	ErrorUnknownType      = errors.New("metric has unknown type")
	ErrorInvalidName      = errors.New("metric has incorrect name")
	ErrorInvalidType      = errors.New("metric has incorrect type")
	ErrorInvalidValue     = errors.New("metric has incorrect value")
	ErrorInvalidJSON      = errors.New("can't convert data to JSON")
	ErrorInternal         = errors.New("internal error storage")
	ErrorInvalidSignature = errors.New("invalid signature metric")
)

// ConvertToHTTP Преобразование ошибки Storage в HTTP код
func ConvertToHTTP(err error) int {
	switch err {
	case ErrorNotFound:
		return http.StatusNotFound
	case ErrorUnknownType:
		return http.StatusNotImplemented
	case
		ErrorInvalidName,
		ErrorInvalidType,
		ErrorInvalidValue,
		ErrorInvalidJSON,
		ErrorInvalidSignature:

		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
