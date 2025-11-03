package app

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	"chat-agent/internal/logging"
	chatpb "chat-agent/proto"
)

// Config captures minimal server configuration options for the skeleton application.
type Config struct {
	GRPCPort int
}

// App owns the lifecycle of the gRPC server.
type App struct {
	cfg        Config
	logger     logging.Logger
	grpcServer *grpc.Server
}

// New creates a new App instance with the provided configuration and service implementation.
func New(cfg Config, logger logging.Logger, chatService chatpb.ChatAgentServiceServer, opts ...grpc.ServerOption) (*App, error) {
	if cfg.GRPCPort == 0 {
		return nil, fmt.Errorf("grpc port must be provided")
	}

	server := grpc.NewServer(opts...)
	chatpb.RegisterChatAgentServiceServer(server, chatService)

	return &App{cfg: cfg, logger: logger, grpcServer: server}, nil
}

// Run starts the gRPC server and blocks until the context is cancelled or a fatal error occurs.
func (a *App) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", a.cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	a.logger.With(logging.Field{Key: "addr", Value: addr}).Info("starting gRPC server")

	var serveErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.grpcServer.Serve(listener); err != nil {
			serveErr = err
		}
	}()

	<-ctx.Done()
	a.grpcServer.GracefulStop()
	wg.Wait()

	if serveErr != nil {
		return serveErr
	}
	return ctx.Err()
}
