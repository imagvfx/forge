package forge

import (
	"errors"

	"github.com/imagvfx/forge/service"
)

// fromServiceError converts a (possible) service error to a forge error.
func fromServiceError(err error) error {
	if errors.As(err, &service.NotFoundError{}) {
		return NotFoundError{err.Error()}
	}
	return err
}

type NotFoundError struct {
	err string
}

func (e NotFoundError) Error() string {
	return e.err
}
