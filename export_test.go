package json

import (
	"github.com/night-codes/go-json/internal/errors"
)

var (
	NewSyntaxError    = errors.ErrSyntax
	NewMarshalerError = errors.ErrMarshaler
)
