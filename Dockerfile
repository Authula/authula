# -------------------
# Build stage
# -------------------

FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata build-base

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the application
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -o server ./cmd/main.go

# -------------------
# production stage
# -------------------

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
  adduser -D -u 1000 -G appgroup appuser

WORKDIR /home/appuser

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/server .

# Set user
USER appuser

# Expose port
EXPOSE 8080

ENV GO_ENV=production

# Run the application
CMD ["./server"]
