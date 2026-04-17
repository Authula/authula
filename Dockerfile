# -------------------
# Build stage
# -------------------
FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o server ./cmd/main.go

# -------------------
# Production stage
# -------------------
FROM alpine:3.23.3

RUN apk add --no-cache ca-certificates curl tzdata

RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appgroup /app/server .

USER appuser

EXPOSE 8080
ENV GO_ENV=production

CMD ["./server"]
