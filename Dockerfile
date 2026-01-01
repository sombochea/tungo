# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w' -o tungo-server cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/tungo-server .

# Expose ports
EXPOSE 5555 8080 9090

# Run the server
ENTRYPOINT ["/app/tungo-server"]
