# -------------------
# Build stage
# -------------------

FROM golang:1.25.5-alpine AS builder

ARG VERSION=dev
ARG BUILD_DATE=unknown

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata build-base

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the application with version info
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
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

# Metadata labels
LABEL maintainer="GoBetterAuth"
LABEL version="${VERSION}"

ENV GO_ENV=production

# Run the application
CMD ["./server"]
