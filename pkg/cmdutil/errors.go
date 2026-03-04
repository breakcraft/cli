package cmdutil

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2/terminal"
)

// FlagErrorf returns a new FlagError that wraps an error produced by
// fmt.Errorf(format, args...).
func FlagErrorf(format string, args ...interface{}) error {
	return FlagErrorWrap(fmt.Errorf(format, args...))
}

// FlagErrorWrap returns a new FlagError that wraps the specified error.
func FlagErrorWrap(err error) error { return &FlagError{err} }

// A *FlagError indicates an error processing command-line flags or other arguments.
// Such errors cause the application to display the usage message.
type FlagError struct {
	// Note: not struct{error}: only *FlagError should satisfy error.
	err error
}

// Error returns the underlying error message.
func (fe *FlagError) Error() string {
	return fe.err.Error()
}

// Unwrap returns the wrapped error.
func (fe *FlagError) Unwrap() error {
	return fe.err
}

// SilentError is an error that triggers exit code 1 without any error messaging
var SilentError = errors.New("SilentError")

// CancelError signals user-initiated cancellation
var CancelError = errors.New("CancelError")

// PendingError signals nothing failed but something is pending
var PendingError = errors.New("PendingError")

// IsUserCancellation reports whether err indicates that the user cancelled the operation.
func IsUserCancellation(err error) bool {
	return errors.Is(err, CancelError) || errors.Is(err, terminal.InterruptErr)
}

// MutuallyExclusive returns a FlagError with the given message if more than one of the conditions is true.
func MutuallyExclusive(message string, conditions ...bool) error {
	numTrue := 0
	for _, ok := range conditions {
		if ok {
			numTrue++
		}
	}
	if numTrue > 1 {
		return FlagErrorf("%s", message)
	}
	return nil
}

// NoResultsError represents an error indicating that a query returned no results.
type NoResultsError struct {
	message string
}

// Error returns the no-results message.
func (e NoResultsError) Error() string {
	return e.message
}

// NewNoResultsError creates a NoResultsError with the given message.
func NewNoResultsError(message string) NoResultsError {
	return NoResultsError{message: message}
}
