package domain

import "errors"

var (
	ErrValidation   = errors.New("sns: validation failed")
	ErrNotFound     = errors.New("sns: resource not found")
	ErrInvalidToken = errors.New("sns: invalid confirmation token")
)
