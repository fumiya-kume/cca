# Use official Go image as base
FROM golang:1.24.3-bookworm

# Install additional tools and dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    unzip \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install development tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install mvdan.cc/gofumpt@latest

# Create a non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get update \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Set up Go environment for the user
RUN mkdir -p /go/pkg && chown -R $USERNAME:$USERNAME /go

# Switch to non-root user
USER $USERNAME

# Set working directory
WORKDIR /workspace

# Configure git for the container
RUN git config --global --add safe.directory /workspace

# Set Go environment variables
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

# Pre-warm Go module cache (optional optimization)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Default command
CMD ["/bin/bash"]