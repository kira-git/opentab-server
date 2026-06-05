package repositories

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict")
var ErrForbidden = errors.New("operation forbidden")
var ErrInvalidState = errors.New("invalid state")
var ErrInvalidRole = errors.New("invalid role")
var ErrInvalidTarget = errors.New("invalid target")
var ErrTokenExpired = errors.New("token expired")
var ErrTokenRevoked = errors.New("token revoked")
var ErrUserDisabled = errors.New("user disabled")
