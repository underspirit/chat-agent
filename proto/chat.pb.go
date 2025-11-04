package chatpb

import (
	"context"
	"errors"
)

type PlayerChatRequest struct {
	PlayerId       string `json:"player_id"`
	PlayerNickname string `json:"player_nickname"`
	NikiId         string `json:"niki_id"`
	NikiName       string `json:"niki_name"`
	InputText      string `json:"input_text"`
}

type PlayerChatResponse struct {
	GeneratedText string `json:"generated_text"`
}

type ChatAgentServiceServer interface {
	Chat(*PlayerChatRequest, ChatAgentService_ChatServer) error
}

type ChatAgentService_ChatServer interface {
	Send(*PlayerChatResponse) error
	Context() context.Context
}

type ChatAgentServiceClient interface {
	Chat(ctx context.Context, in *PlayerChatRequest, handler func(*PlayerChatResponse) error) error
}

type UnimplementedChatAgentServiceServer struct{}

func (UnimplementedChatAgentServiceServer) Chat(*PlayerChatRequest, ChatAgentService_ChatServer) error {
	return errors.New("ChatAgentService.Chat is not implemented")
}
