# ---------- development (Air hot-reload) ----------
FROM golang:1.26-alpine AS development
# poppler-utils: pratinjau halaman-1 PDF (job worker pdf-preview).
RUN apk add --no-cache git curl wget poppler-utils && go install github.com/air-verse/air@latest
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

# Runtime: alpine (bukan distroless) karena worker memerlukan pdftoppm untuk
# pratinjau PDF. Dijalankan sebagai user non-root; hanya satu port internal.
FROM alpine:3.22 AS production
# postgresql-client: pg_dump untuk endpoint backup manual (system.backup).
# poppler-utils: pratinjau halaman-1 PDF (job worker pdf-preview).
RUN apk add --no-cache poppler-utils postgresql16-client ca-certificates tzdata \
  && addgroup -S app && adduser -S -G app app \
  # Direktori backup: fresh named volume mewarisi kepemilikan dari image,
  # jadi user non-root tetap bisa menulis ke dalamnya.
  && mkdir -p /var/backups/grafikarsa && chown -R app:app /var/backups/grafikarsa
COPY --from=builder /api /api
COPY --from=builder /migrate /migrate
COPY --from=builder /worker /worker
COPY --from=builder /seed /seed
USER app
EXPOSE 8080
ENTRYPOINT ["/api"]
