package service

import "errors"

var ErrNotFound = errors.New("not found")

var ErrValidation = errors.New("validation error")

var ErrConflict = errors.New("conflict")

var ErrUnauthorized = errors.New("unauthorized")
