// Package errs
package errs

import "errors"

var (
	ErrDB              = errors.New("database error")
	ErrRecordingWNC    = errors.New("recording wasn't created")
	ErrRecordingWND    = errors.New("recording wasn't deleted")
	ErrRecordingWNF    = errors.New("recording wasn't found")
	ErrDuplicated      = errors.New("object has already existed")
	ErrTokenTTL        = errors.New("token's ttl is over")
	ErrSignToken       = errors.New("failed to sign token")
	ErrEmptyAuthHeader = errors.New("authorization header is required")
	ErrInvalidToken    = errors.New("token is not valid")
)
