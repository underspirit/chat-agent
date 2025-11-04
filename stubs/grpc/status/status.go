package status

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

func Errorf(_ codes.Code, format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
