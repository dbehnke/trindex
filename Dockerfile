# Build stage with Node.js for web UI
FROM golang:1.26-alpine AS builder

# Install Node.js and npm
RUN apk add --no-cache nodejs npm

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (including web/)
COPY . .

# Build web UI
WORKDIR /app/web
RUN npm ci && npm run build

# Copy dist to internal/web for embedding
WORKDIR /app
RUN cp -r web/dist internal/web/

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
