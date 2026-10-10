# ---------- development (Air hot-reload) ----------
FROM golang:1.26-alpine AS development
RUN apk add --no-cache git curl wget && go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ---------- production ----------
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /api ./cmd/api \
  && CGO_ENABLED=0 go build -ldflags="-s -w" -o /migrate ./cmd/tools/migrate \
  && CGO_ENABLED=0 go build -ldflags="-s -w" -o /worker ./cmd/worker \
  && CGO_ENABLED=0 go build -ldflags="-s -w" -o /seed ./cmd/tools/seed

FROM gcr.io/distroless/static AS production
COPY --from=builder /api /api
COPY --from=builder /migrate /migrate
COPY --from=builder /worker /worker
COPY --from=builder /seed /seed
EXPOSE 8080
ENTRYPOINT ["/api"]
