# Chat Agent

This repository hosts a gRPC based assistant service with a minimal CLI client for manual testing. The implementation follows the high level architecture described in `system-design.md`.

## Prerequisites
- Go 1.22+
- (Optional) An OpenAI compatible API endpoint if you want to exercise real LLM calls. Tests run against the local mock and do not need network access.

## Configuration
The service is driven by environment variables. All variables are optional unless otherwise stated.

| Variable | Description | Default |
| --- | --- | --- |
| `CHAT_AGENT_GRPC_PORT` | Port that the gRPC server listens on. | `50051` |
| `CHAT_AGENT_DATA_DIR` | Directory used for persisting conversation history. | `./data` |
| `OPENAI_API_KEY` | API key used when calling the OpenAI chat endpoint. | _required for real requests_ |
| `OPENAI_BASE_URL` | Base URL for the OpenAI compatible API endpoint. | `https://api.openai.com/v1` |
| `OPENAI_MODEL` | Model identifier used for prompt construction and requests. | `gpt-4o-mini` |

## Running the gRPC service
```bash
export CHAT_AGENT_DATA_DIR="./data"
export OPENAI_API_KEY="sk-..."          # required only for real OpenAI calls
export OPENAI_BASE_URL="https://api.openai.com" # optional override
export OPENAI_MODEL="gpt-4o-mini"        # optional override

go run ./cmd/chat-agent
```

The server listens for streaming requests on `CHAT_AGENT_GRPC_PORT` and persists the conversation history in the configured `CHAT_AGENT_DATA_DIR`.

## CLI test client
A small command line client is provided to exercise the gRPC API.

```bash
go run ./cmd/chat-client \
  -addr 127.0.0.1:50051 \
  -player player-1 \
  -nickname "Player One" \
  -niki niki-1 \
  -nikiName "Niki" \
  -input "Hello!"
```

The client will stream the assistant response to stdout.

## Running tests
```bash
go test ./...
```

The test suite uses the in-repo gRPC and OpenAI mocks so no external dependencies are required.
