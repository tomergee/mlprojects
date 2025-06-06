# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod first to leverage Docker cache for dependencies
COPY go.mod ./
# Create go.sum if it doesn't exist and ensure dependencies are clean
RUN go mod tidy
# Download dependencies (go.sum will be used if it exists)
RUN go mod download && go mod verify

# Copy the rest of the application source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static linking, GOOS=linux for Linux environment
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /mcp_server .

# Stage 2: Create the runtime image
FROM alpine:latest

WORKDIR /app

# Install kubectl
# Find the latest stable kubectl version for amd64
RUN apk add --no-cache curl \
    && KUBECTL_LATEST_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt) \
    && echo "Downloading kubectl version ${KUBECTL_LATEST_VERSION}..." \
    && curl -LO "https://dl.k8s.io/release/${KUBECTL_LATEST_VERSION}/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/kubectl \
    && apk del curl

# Copy the compiled binary from the builder stage
COPY --from=builder /mcp_server /app/mcp_server

# Expose the port the server listens on
EXPOSE 8080

# Set the command to run the server executable
CMD ["/app/mcp_server"]
