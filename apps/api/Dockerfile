FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# The worker is the only binary that links libvips. Keep this builder and the
# worker runtime separate from the static API image.
FROM golang:1.26-trixie AS worker-builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libvips-dev \
        pkg-config \
        poppler-utils \
        libheif-plugin-aomenc \
        libheif-plugin-dav1d \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM debian:trixie-slim AS worker

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libvips42t64 \
        libheif-plugin-aomenc \
        libheif-plugin-dav1d \
        poppler-utils \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 worker \
    && useradd --system --uid 10001 --gid worker --home-dir /nonexistent --shell /usr/sbin/nologin worker

COPY --from=worker-builder /out/worker /worker

USER 10001:10001
EXPOSE 8092
ENTRYPOINT ["/worker"]

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/api /api
COPY --from=builder /out/admin /admin
COPY --from=builder /out/migrate /migrate

USER nonroot:nonroot
EXPOSE 8091
ENTRYPOINT ["/api"]
