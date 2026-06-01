package service

import "errors"

var ErrNotFound = errors.New("user not found")

var ErrValidation = errors.New("validation error")
