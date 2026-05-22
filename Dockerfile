# Build stage
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.20
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.3 && \
    $(go env GOPATH)/bin/swag init -g cmd/server/main.go -o docs

# Build the application
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0
ARG BUILD_TAGS=""
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED}
RUN go build -a -installsuffix cgo -tags "${BUILD_TAGS}" -o /app/server ./cmd/server

# Final stage
FROM alpine:${ALPINE_VERSION}

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

ARG APP_PORT=8080
EXPOSE ${APP_PORT}

CMD ["./server"]
