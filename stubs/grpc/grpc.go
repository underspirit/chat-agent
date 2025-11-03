package grpc

import (
	"context"
	"errors"
	"io"
	"net"
)

type Server struct{}

type ServerOption interface{}

type ServiceRegistrar interface {
	RegisterService(*ServiceDesc, interface{})
}

type ServiceDesc struct {
	ServiceName string
	HandlerType interface{}
	Methods     []MethodDesc
	Streams     []StreamDesc
	Metadata    interface{}
}

type MethodDesc struct{}

type StreamDesc struct {
	StreamName    string
	Handler       func(srv interface{}, stream ServerStream) error
	ServerStreams bool
}

type ServerStream interface {
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

type ClientStream interface {
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
	CloseSend() error
}

type ClientConnInterface interface {
	NewStream(ctx context.Context, desc *StreamDesc, method string, opts ...CallOption) (ClientStream, error)
}

type CallOption interface{}

type DialOption interface{}

type TransportCredentials interface{}

type ClientConn struct{}

type RegisterableServer interface {
	RegisterService(desc *ServiceDesc, impl interface{})
}

func NewServer(opts ...ServerOption) *Server {
	return &Server{}
}

func (s *Server) RegisterService(_ *ServiceDesc, _ interface{}) {}

func (s *Server) Serve(_ net.Listener) error { return nil }

func (s *Server) GracefulStop() {}

func Dial(_ string, _ ...DialOption) (*ClientConn, error) {
	return &ClientConn{}, nil
}

func (c *ClientConn) Close() error { return nil }

func (c *ClientConn) NewStream(ctx context.Context, _ *StreamDesc, _ string, _ ...CallOption) (ClientStream, error) {
	return &noopClientStream{ctx: ctx}, nil
}

func WithTransportCredentials(_ TransportCredentials) DialOption { return nil }

type noopClientStream struct {
	ctx  context.Context
	sent bool
}

func (s *noopClientStream) Context() context.Context { return s.ctx }

func (s *noopClientStream) SendMsg(_ interface{}) error {
	s.sent = true
	return nil
}

func (s *noopClientStream) RecvMsg(m interface{}) error {
	if !s.sent {
		return errors.New("no request sent")
	}
	switch msg := m.(type) {
	case *struct{}:
		return nil
	default:
		_ = msg
	}
	return io.EOF
}

func (s *noopClientStream) CloseSend() error { return nil }
