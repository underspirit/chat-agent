package insecure

import "google.golang.org/grpc"

type insecureCreds struct{}

// NewCredentials returns a placeholder transport credentials implementation.
func NewCredentials() grpc.TransportCredentials {
	return insecureCreds{}
}
