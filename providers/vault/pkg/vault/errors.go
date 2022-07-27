package vault

import (
	"fmt"
)

// Return this error when get HTTP code 403.
type forbiddenError struct {
	error
}

// Error string representation of the error.
func (e *forbiddenError) Error() string {
	return fmt.Sprintf("forbidden error %s", e.error)
}

func newForbiddenError(err error) error {
	return &forbiddenError{error: err}
}
