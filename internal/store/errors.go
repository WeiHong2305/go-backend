package store

import "errors"

// ErrNotFound means the requested user does not exist.
// Handlers should use errors.Is(err, store.ErrNotFound) to return 404 vs 500.
var ErrNotFound = errors.New("user not found")
