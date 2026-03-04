package auth

import (
	"errors"

	ghauth "github.com/cli/go-gh/v2/pkg/auth"
)

// ErrUnsupportedHost is returned when the host is not supported for attestation operations.
var ErrUnsupportedHost = errors.New("An unsupported host was detected. Note that gh attestation does not currently support GHES")

// IsHostSupported returns an error if the given host is an unsupported enterprise server.
func IsHostSupported(host string) error {
	if ghauth.IsEnterprise(host) {
		return ErrUnsupportedHost
	}
	return nil
}
