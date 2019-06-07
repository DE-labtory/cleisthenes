package bba

import (
	"errors"
)

var ErrInvalidType = errors.New("request type is invalid")
var ErrNoResult = errors.New("no result with id")

func IsErrNoResult(err error) bool {
	return err == ErrNoResult
}
