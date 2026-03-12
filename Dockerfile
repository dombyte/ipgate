# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy and download dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN GOAMD64=v3 go build -o /ipgate ./cmd/ipgate

FROM alpine:latest as certs
RUN apk --update add ca-certificates


FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
ARG TARGETPLATFORM
COPY --from=builder /ipgate /app
COPY --from=ghcr.io/tarampampam/microcheck:1 /bin/httpcheck /bin/httpcheck
ENTRYPOINT ["/app/ipgate"]
