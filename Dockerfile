# syntax=docker/dockerfile:1

# Build stage — compile the static binary.
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /cure ./cmd/cure

# Final stage — minimal runtime image.
# alpine (not scratch) provides: shell for Job command templates,
# ca-certificates for HTTPS in cure trace http, and /etc/passwd for non-root user.
FROM alpine:3
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 1000 cure
COPY --from=builder /cure /usr/local/bin/cure
USER cure
ENTRYPOINT ["cure"]
