package bba

import (
	"errors"
)

var ErrUndefinedRequestType = errors.New("unexpected request type")
var ErrInvalidType = errors.New("request type is invalid")
var ErrNoResult = errors.New("no result with id")

func IsErrNoResult(err error) bool {
	return err == ErrNoResult
}
