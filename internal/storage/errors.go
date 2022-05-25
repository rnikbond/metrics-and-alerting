package storage

import (
	"errors"
	"net/http"
)

// Ошибки метрики
var (
	ErrorNotFound         = errors.New("metric not found")
	ErrorUnknownType      = errors.New("metric has unknown type")
	ErrorInvalidID        = errors.New("metric has incorrect id")
	ErrorInvalidType      = errors.New("metric has incorrect type")
	ErrorInvalidValue     = errors.New("metric has incorrect value")
	ErrorInvalidJSON      = errors.New("can't convert data JSON to metric")
	ErrorInvalidSignature = errors.New("invalid signature metric")
)

// Ошибки внешнего хранилища
var (
	ErrorInvalidFilePath = errors.New("invalid path to file storage")
	ErrorExternalStorage = errors.New("internal error external storage")
	ErrorDatabaseDriver  = errors.New("database driver not initialized")
)

// ErrorHTTP - Преобразование ошибки Storage в HTTP код
func ErrorHTTP(err error) int {
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
		ErrorInvalidSignature:

		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
