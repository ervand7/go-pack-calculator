package calculator

import "errors"

var (
	ErrInvalidItemCount = errors.New("item count must be zero or greater")
	ErrNoPackSizes      = errors.New("at least one pack size is required")
	ErrInvalidPackSize  = errors.New("pack sizes must be positive")
)
