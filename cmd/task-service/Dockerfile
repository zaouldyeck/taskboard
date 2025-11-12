# Multi-stage builder.

# BUILD STAGE
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Static binary with debug info stripped.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o task-service ./cmd/task-service


# RUNTIME STAGE
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /app/task-service /task-service

EXPOSE 50051

ENTRYPOINT ["/task-service"]
