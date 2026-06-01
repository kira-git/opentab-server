package repositories

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict")
var ErrForbidden = errors.New("operation forbidden")
