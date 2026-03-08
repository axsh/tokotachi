FROM golang:1.23-bookworm

# Install Docker CLI (for testing container operations)
RUN apt-get update && apt-get install -y \
    docker.io \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

WORKDIR /workspace

CMD ["sleep", "infinity"]
