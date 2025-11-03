module chat-agent

go 1.25.3

require (
    google.golang.org/grpc v1.64.0
)

replace google.golang.org/grpc => ./stubs/grpc

