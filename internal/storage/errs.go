package storage

import (
	"errors"
	"net/http"
)

// Ошибки метрики
var (
	ErrNotFound     = errors.New("metric not found")
	ErrUnknownType  = errors.New("metric has unknown type")
	ErrInvalidID    = errors.New("metric has incorrect id")
	ErrInvalidType  = errors.New("metric has incorrect type")
	ErrInvalidValue = errors.New("metric has incorrect value")
	ErrInvalidJSON  = errors.New("can't convert data JSON to metric")
	ErrSignFailed   = errors.New("sign verification failed")
)

// Ошибки внешнего хранилища
var (
	ErrInvalidFilePath  = errors.New("invalid path to file storage")
	ErrInvalidDSN       = errors.New("invalid data source name")
	ErrFailedConnection = errors.New("can not create connection")
)

// ErrorHTTP - Преобразование ошибки Storage в HTTP код
func ErrorHTTP(err error) int {

	if errUnwrap := errors.Unwrap(err); errUnwrap != nil {
		err = errUnwrap
	}

	switch err {
	case ErrNotFound:
		return http.StatusNotFound

	case ErrUnknownType:
		return http.StatusNotImplemented

	case
		ErrInvalidID,
		ErrInvalidType,
		ErrInvalidValue,
		ErrInvalidJSON,
		ErrSignFailed:

		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
