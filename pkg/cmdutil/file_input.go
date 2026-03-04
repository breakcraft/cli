package cmdutil

import (
	"io"
	"os"
)

// ReadFile reads the contents of filename, or from stdin if filename is "-".
func ReadFile(filename string, stdin io.ReadCloser) ([]byte, error) {
	if filename == "-" {
		b, err := io.ReadAll(stdin)
		_ = stdin.Close()
		return b, err
	}

	return os.ReadFile(filename)
}
