package storage

import (
	"errors"
	"net/http"
)

// Ошибки метрики
var (
	ErrorNotFound     = errors.New("metric not found")
	ErrorUnknownType  = errors.New("metric has unknown type")
	ErrorInvalidID    = errors.New("metric has incorrect id")
	ErrorInvalidType  = errors.New("metric has incorrect type")
	ErrorInvalidValue = errors.New("metric has incorrect value")
	ErrorInvalidJSON  = errors.New("can't convert data JSON to metric")
	ErrorSignFailed   = errors.New("sign verification failed")
)

// Ошибки внешнего хранилища
var (
	ErrorInvalidFilePath  = errors.New("invalid path to file storage")
	ErrorInvalidDSN       = errors.New("invalid data source name")
	ErrorFailedConnection = errors.New("can not create connection")
)

// ErrorHTTP - Преобразование ошибки Storage в HTTP код
func ErrorHTTP(err error) int {

	if errUnwrap := errors.Unwrap(err); errUnwrap != nil {
		err = errUnwrap
	}

	switch err {
	case ErrorNotFound:
		return http.StatusNotFound

	case ErrorUnknownType:
		return http.StatusNotImplemented

	case
		ErrorInvalidID,
		ErrorInvalidType,
		ErrorInvalidValue,
		ErrorInvalidJSON,
		ErrorSignFailed:

		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
