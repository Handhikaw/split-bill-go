# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux

# Cache dependency layer separately from source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/split-bill-go ./cmd/api

# --- runtime stage ---
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/split-bill-go /app/split-bill-go

USER app

ENV APP_ENV=production \
    APP_PORT=8080 \
    TZ=Asia/Jakarta

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/split-bill-go"]
