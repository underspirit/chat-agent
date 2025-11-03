package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	chatpb "chat-agent/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "gRPC server address")
	playerID := flag.String("player", "player-1", "player id")
	playerNickname := flag.String("nickname", "Player One", "player nickname")
	nikiID := flag.String("niki", "niki-1", "niki id")
	nikiName := flag.String("nikiName", "Niki", "niki display name")
	input := flag.String("input", "Hello!", "message to send")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := chatpb.NewChatAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream, err := client.Chat(ctx, &chatpb.PlayerChatRequest{
		PlayerId:       *playerID,
		PlayerNickname: *playerNickname,
		NikiId:         *nikiID,
		NikiName:       *nikiName,
		InputText:      *input,
	})
	if err != nil {
		log.Fatalf("chat call: %v", err)
	}

	fmt.Printf("Streaming response from %s...\n", *addr)
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			if err == context.Canceled || err == context.DeadlineExceeded {
				log.Fatalf("stream cancelled: %v", err)
			}
			log.Fatalf("stream error: %v", err)
		}
		fmt.Print(resp.GeneratedText)
	}
	fmt.Println()
}
