# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags='-s -w' -o /usr/local/bin/whatsapp-api .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

RUN addgroup -S app && adduser -S app -G app

RUN mkdir -p /data/store && chown app:app /data/store

COPY --from=builder /usr/local/bin/whatsapp-api /usr/local/bin/whatsapp-api

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["whatsapp-api"]
CMD ["serve"]
