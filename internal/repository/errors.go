package repository

import "errors"

var ErrNotFound = errors.New("not found")

var ErrDuplicateEmail = errors.New("email already exists")
