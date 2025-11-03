package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	chatpb "chat-agent/proto"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "address of the ChatAgent gRPC server")
	player := flag.String("player", "player-1", "player id")
	playerName := flag.String("player-name", "Player", "player nickname")
	niki := flag.String("niki", "niki-1", "niki id")
	nikiName := flag.String("niki-name", "Niki", "niki name")
	message := flag.String("message", "Hello there!", "message to send")
	timeout := flag.Duration("timeout", 10*time.Second, "request timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, *addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial %s: %v", *addr, err)
	}
	defer conn.Close()

	client := chatpb.NewChatAgentServiceClient(conn)
	stream, err := client.Chat(ctx, &chatpb.PlayerChatRequest{PlayerId: *player, PlayerNickname: *playerName, NikiId: *niki, NikiName: *nikiName, InputText: *message})
	if err != nil {
		log.Fatalf("chat call failed: %v", err)
	}

	fmt.Println("--- ChatAgent Response ---")
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("stream recv failed: %v", err)
		}
		fmt.Print(resp.GeneratedText)
	}
	fmt.Println()
}
