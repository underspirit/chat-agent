package insecure

import "google.golang.org/grpc"

type insecureCreds struct{}

func NewCredentials() grpc.TransportCredentials {
	return insecureCreds{}
}

func (insecureCreds) Info() interface{} { return nil }
