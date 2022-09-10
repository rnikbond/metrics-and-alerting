package errs

import (
	"errors"
	"net/http"
)

type ErrStorage struct {
	Value string
}

func NewErr(s string) ErrStorage {
	return ErrStorage{Value: s}
}

func (es ErrStorage) Error() string {
	return es.Value
}

// Ошибки метрики
var (
	ErrNotFound     = NewErr("metric not found")
	ErrUnknownType  = NewErr("metric has unknown type")
	ErrInvalidID    = NewErr("metric has incorrect id")
	ErrInvalidType  = NewErr("metric has incorrect type")
	ErrInvalidValue = NewErr("metric has incorrect value")
	ErrInvalidJSON  = NewErr("can't convert data JSON to metric")
	ErrSignFailed   = NewErr("sign verification failed")
)

// Ошибки внешнего хранилища
var (
	ErrInvalidFilePath  = NewErr("invalid path to fileStorage storage")
	ErrInvalidDSN       = NewErr("invalid data source name")
	ErrFailedConnection = NewErr("can not create connection")
)

// ErrorHTTP - Преобразование ошибки Storage в HTTP код
func ErrorHTTP(err error) int {

	var storeErr ErrStorage
	if !errors.As(err, &storeErr) {
		return http.StatusInternalServerError
	}

	switch storeErr {
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
