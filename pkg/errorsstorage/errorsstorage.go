package errorsstorage

import (
	"errors"
	"net/http"
)

var (
	ErrorNotFound       = errors.New("metric not found")
	ErrorUnknownType    = errors.New("metric has unknown type")
	ErrorIncorrectName  = errors.New("metric has incorrect name")
	ErrorIncorrectValue = errors.New("metric has incorrect value")
)

// ConvertToHTTP Преобразование ошибки Storage в HTTP код
func ConvertToHTTP(err error) int {
	switch err {
	case ErrorNotFound:
		return http.StatusNotFound
	case ErrorUnknownType:
		return http.StatusNotImplemented
	case ErrorIncorrectValue:
		return http.StatusBadRequest
	}

	return http.StatusOK
}
