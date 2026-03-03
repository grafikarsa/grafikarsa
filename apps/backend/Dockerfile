# ==============================================
# GRAFIKARSA BACKEND - Multi-stage Dockerfile
# ==============================================
# Stages: development, builder, production

# ---- Development stage (with Air hot-reload) ----
FROM golang:1.24-alpine AS development

WORKDIR /app

# Install Air for hot-reload and git for dependencies
RUN apk add --no-cache git \
    && go install github.com/air-verse/air@latest

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Run with Air for hot-reload
CMD ["air", "-c", ".air.toml"]

# ---- Builder stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# ---- Production stage ----
FROM alpine:latest AS production

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
