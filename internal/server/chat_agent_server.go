package server

import (
	"context"

	"chat-agent/internal/conversation"
	"chat-agent/internal/logging"
	chatpb "chat-agent/proto"
)

// ChatAgentGRPCServer implements the gRPC service defined in proto/chat.proto.
type ChatAgentGRPCServer struct {
	chatpb.UnimplementedChatAgentServiceServer

	manager conversation.Manager
	logger  logging.Logger
}

// NewChatAgentGRPCServer wires a conversation manager into the gRPC surface.
func NewChatAgentGRPCServer(manager conversation.Manager, logger logging.Logger) *ChatAgentGRPCServer {
	return &ChatAgentGRPCServer{manager: manager, logger: logger}
}

// Chat handles the PlayerChatRequest using the conversation manager and streams responses back to the caller.
func (s *ChatAgentGRPCServer) Chat(req *chatpb.PlayerChatRequest, stream chatpb.ChatAgentService_ChatServer) error {
	adapter := &streamAdapter{stream: stream}
	ctx := stream.Context()

	chatReq := conversation.ChatRequest{
		PlayerID:       req.PlayerId,
		PlayerNickname: req.PlayerNickname,
		NikiID:         req.NikiId,
		NikiName:       req.NikiName,
		InputText:      req.InputText,
	}

	s.logger.With(
		logging.Field{Key: "player_id", Value: req.PlayerId},
		logging.Field{Key: "niki_id", Value: req.NikiId},
	).Info("grpc Chat invoked")

	return s.manager.HandleChat(ctx, chatReq, adapter)
}

// streamAdapter bridges the conversation.Stream interface with the generated gRPC stream.
type streamAdapter struct {
	stream chatpb.ChatAgentService_ChatServer
}

func (a *streamAdapter) SendChunk(ctx context.Context, chunk conversation.ChatChunk) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return a.stream.Send(&chatpb.PlayerChatResponse{GeneratedText: chunk.Text})
}

var _ conversation.Stream = (*streamAdapter)(nil)
