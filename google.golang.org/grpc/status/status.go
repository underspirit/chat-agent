package status

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// Errorf creates an error with the provided status code.
func Errorf(code codes.Code, format string, args ...interface{}) error {
	return fmt.Errorf("grpc status %d: %s", code, fmt.Sprintf(format, args...))
}
