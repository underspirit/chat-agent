package grpc

import (
	"context"
	"errors"
	"net"
)

type CallOption interface{}

type DialOption interface{}

type ServerOption interface{}

type TransportCredentials interface{}

type StreamDesc struct {
	StreamName    string
	Handler       func(srv interface{}, stream ServerStream) error
	ServerStreams bool
	ClientStreams bool
}

type MethodDesc struct {
	MethodName string
	Handler    func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor interface{}) (interface{}, error)
}

type ServiceDesc struct {
	ServiceName string
	HandlerType interface{}
	Methods     []MethodDesc
	Streams     []StreamDesc
	Metadata    interface{}
}

type ClientStream interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
	CloseSend() error
	Context() context.Context
}

type ServerStream interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
	Context() context.Context
}

type ClientConnInterface interface {
	NewStream(ctx context.Context, desc *StreamDesc, method string, opts ...CallOption) (ClientStream, error)
}

type ServiceRegistrar interface {
	RegisterService(desc *ServiceDesc, impl interface{})
}

type Server struct{}

func NewServer(opts ...ServerOption) *Server {
	return &Server{}
}

func (s *Server) Serve(lis net.Listener) error {
	return errors.New("grpc Serve not implemented in stub")
}

func (s *Server) GracefulStop() {}

func (s *Server) Stop() {}

func (s *Server) RegisterService(desc *ServiceDesc, impl interface{}) {}

func RegisterService(s ServiceRegistrar, desc *ServiceDesc, impl interface{}) {
	s.RegisterService(desc, impl)
}

type ClientConn struct{}

func DialContext(ctx context.Context, target string, opts ...DialOption) (*ClientConn, error) {
	return &ClientConn{}, nil
}

func (c *ClientConn) NewStream(ctx context.Context, desc *StreamDesc, method string, opts ...CallOption) (ClientStream, error) {
	return nil, errors.New("grpc ClientConn stream not implemented in stub")
}

func (c *ClientConn) Close() error {
	return nil
}

func WithTransportCredentials(creds TransportCredentials) DialOption {
	return nil
}

func WithContextDialer(dialer func(context.Context, string) (net.Conn, error)) DialOption {
	return nil
}
