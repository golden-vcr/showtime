package imagegen

import (
	"errors"
	"fmt"
)

// ErrRejected is returned when the image generation API rejected the request to
// generate one or more images, typically because the prompt contained text that was
// classified as objectionable
var ErrRejected = errors.New("image generation request rejected")

// rejectionError unwraps to ErrRejected and carries the original client-facing message
// returned as a 400 response from the image generation API
type rejectionError struct {
	message string
}

// Error formats a rejection error, prefixed with the ErrRejected message and including
// the original error message received from the image generation API
func (e *rejectionError) Error() string {
	return fmt.Sprintf("%v: %s", ErrRejected, e.message)
}

// Unwrap identifies a value of this type as synonymous with ErrRejected
func (e *rejectionError) Unwrap() error {
	return ErrRejected
}
