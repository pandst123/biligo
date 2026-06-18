# syntax=docker/dockerfile:1

FROM node:24-alpine AS web-builder
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.22-alpine AS go-builder
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build \
    -tags embed_web \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/biligo \
    ./cmd/server

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S biligo \
    && adduser -S -G biligo biligo \
    && mkdir -p /app/data /app/logs \
    && chown -R biligo:biligo /app
COPY --from=go-builder /out/biligo /app/biligo
USER biligo
ENV BILIGO_CONFIG=/app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/biligo"]
