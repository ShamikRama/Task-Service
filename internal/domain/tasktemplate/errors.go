package tasktemplate

import "errors"

var (
	ErrNotFound     = errors.New("template not found")
	ErrInvalidInput = errors.New("invalid template input")
)
