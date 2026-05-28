# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends protobuf-compiler \
	&& rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 \
	&& go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN protoc -I api/proto \
	--go_out=./api/gen --go_opt=paths=source_relative \
	--go-grpc_out=./api/gen --go-grpc_opt=paths=source_relative \
	api/proto/agent/v1/agent.proto

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/agent-grpc ./cmd/agent-grpc/

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	chromium \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/agent-grpc ./agent-grpc

ENV GRPC_LISTEN_ADDR=:3333

EXPOSE 3333

VOLUME ["/app/writes", "/app/log"]

CMD ["./agent-grpc"]
