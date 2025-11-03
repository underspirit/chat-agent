// Code generated manually for initial skeleton; DO NOT EDIT.
// In future iterations, regenerate via protoc once toolchain is available.
package chatpb

import (
	context "context"
	reflect "reflect"
	sync "sync"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// PlayerChatRequest mirrors the proto definition for chat.PlayerChatRequest.
type PlayerChatRequest struct {
	PlayerId       string
	PlayerNickname string
	NikiId         string
	NikiName       string
	InputText      string
}

// PlayerChatResponse mirrors the proto definition for chat.PlayerChatResponse.
type PlayerChatResponse struct {
	GeneratedText string
}

// ChatAgentServiceClient is the client API for ChatAgentService service.
type ChatAgentServiceClient interface {
	Chat(ctx context.Context, in *PlayerChatRequest, opts ...grpc.CallOption) (ChatAgentService_ChatClient, error)
}

type chatAgentServiceClient struct {
	cc grpc.ClientConnInterface
}

// NewChatAgentServiceClient constructs a new gRPC client.
func NewChatAgentServiceClient(cc grpc.ClientConnInterface) ChatAgentServiceClient {
	return &chatAgentServiceClient{cc}
}

func (c *chatAgentServiceClient) Chat(ctx context.Context, in *PlayerChatRequest, opts ...grpc.CallOption) (ChatAgentService_ChatClient, error) {
	stream, err := c.cc.NewStream(ctx, &ChatAgentService_ServiceDesc.Streams[0], ChatAgentService_Chat_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &chatAgentServiceChatClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// ChatAgentService_ChatClient exposes a streaming response reader.
type ChatAgentService_ChatClient interface {
	Recv() (*PlayerChatResponse, error)
	grpc.ClientStream
}

type chatAgentServiceChatClient struct {
	grpc.ClientStream
}

func (x *chatAgentServiceChatClient) Recv() (*PlayerChatResponse, error) {
	m := new(PlayerChatResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChatAgentServiceServer defines the server API.
type ChatAgentServiceServer interface {
	Chat(*PlayerChatRequest, ChatAgentService_ChatServer) error
	mustEmbedUnimplementedChatAgentServiceServer()
}

// UnimplementedChatAgentServiceServer can be embedded to have forward compatible implementations.
type UnimplementedChatAgentServiceServer struct{}

func (UnimplementedChatAgentServiceServer) Chat(*PlayerChatRequest, ChatAgentService_ChatServer) error {
	return status.Errorf(codes.Unimplemented, "method Chat not implemented")
}
func (UnimplementedChatAgentServiceServer) mustEmbedUnimplementedChatAgentServiceServer() {}

// UnsafeChatAgentServiceServer may be embedded to opt out of forward compatibility.
type UnsafeChatAgentServiceServer interface {
	mustEmbedUnimplementedChatAgentServiceServer()
}

// RegisterChatAgentServiceServer registers the service implementation with the provided registrar.
func RegisterChatAgentServiceServer(s grpc.ServiceRegistrar, srv ChatAgentServiceServer) {
	s.RegisterService(&ChatAgentService_ServiceDesc, srv)
}

// ChatAgentService_ChatServer represents the server stream for Chat.
type ChatAgentService_ChatServer interface {
	Send(*PlayerChatResponse) error
	grpc.ServerStream
}

type chatAgentServiceChatServer struct {
	grpc.ServerStream
}

func (x *chatAgentServiceChatServer) Send(m *PlayerChatResponse) error {
	return x.ServerStream.SendMsg(m)
}

const (
	// ChatAgentService_Chat_FullMethodName is the fully-qualified gRPC method name.
	ChatAgentService_Chat_FullMethodName = "/chat.ChatAgentService/Chat"
)

func _ChatAgentService_Chat_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(PlayerChatRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ChatAgentServiceServer).Chat(m, &chatAgentServiceChatServer{stream})
}

// ChatAgentService_ServiceDesc describes the ChatAgentService service definition.
var ChatAgentService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "chat.ChatAgentService",
	HandlerType: (*ChatAgentServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Chat",
			Handler:       _ChatAgentService_Chat_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/chat.proto",
}

// The following variables mirror generated file metadata to simplify future protoc adoption.
var (
	file_proto_chat_proto_rawDescOnce sync.Once
	file_proto_chat_proto_rawDescData = []byte("placeholder")
)

// File_proto_chat_proto returns a placeholder descriptor to satisfy future usages.
func File_proto_chat_proto() []byte {
	file_proto_chat_proto_rawDescOnce.Do(func() {})
	return file_proto_chat_proto_rawDescData
}

// Reference imports to suppress unused warnings once real code generation is in place.
func init() {
	_ = context.Background
	_ = reflect.TypeOf
}
