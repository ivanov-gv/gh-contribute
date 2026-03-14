package auth

import "errors"

var (
	// ErrNotAuthenticated is returned when no token is found in env or config file.
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrTokenInvalid is returned when GitHub rejects the token with 401.
	ErrTokenInvalid = errors.New("token invalid or expired")
)
