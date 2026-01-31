# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/serverguard-api ./cmd/serverguard-api

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Copy binary
COPY --from=builder /app/bin/serverguard-api .

# Copy migrations
COPY --from=builder /app/internal/database/migrations ./migrations

# Create non-root user
RUN adduser -D -H -s /sbin/nologin serverguard
USER serverguard

EXPOSE 8080

CMD ["./serverguard-api"]
