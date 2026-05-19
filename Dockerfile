FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags='-s -w' \
    -o /out/notify-bot ./cmd/bot

FROM alpine:3.23
RUN apk upgrade --no-cache \
 && apk add --no-cache ca-certificates tzdata \
 && rm -rf /var/cache/apk/*

COPY --from=builder /out/notify-bot /usr/local/bin/notify-bot

RUN addgroup -g 10001 app \
 && adduser -D -u 10001 -G app -s /sbin/nologin app

USER app:app
ENTRYPOINT ["/usr/local/bin/notify-bot"]
