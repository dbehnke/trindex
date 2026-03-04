# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o trindex ./cmd/trindex

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/trindex .

# Run as non-root user for security
RUN adduser -D -s /bin/sh trindex
USER trindex

ENTRYPOINT ["./trindex"]
