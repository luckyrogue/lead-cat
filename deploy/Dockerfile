FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@9 --activate && pnpm install --frozen-lockfile
COPY frontend .

ARG VITE_AUTH_DEV_MODE=false
ENV VITE_AUTH_DEV_MODE=${VITE_AUTH_DEV_MODE}

RUN pnpm build

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/lead-cat ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /out/lead-cat /app/lead-cat
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /src/migrations /app/migrations
COPY --from=frontend /app/dist /app/web/dist
WORKDIR /app
ENV STATIC_DIR=/app/web/dist
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/lead-cat"]
