FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /usr/bin/server ./cmd/server/main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install certificates
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /usr/bin/server /usr/bin/server

EXPOSE 8080

CMD ["/usr/bin/server"]
