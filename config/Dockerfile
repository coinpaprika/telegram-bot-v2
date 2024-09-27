FROM golang:1.23.1 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot ./cmd

FROM alpine:3.18 AS curl-stage

RUN apk update && \
    apk add --no-cache curl ca-certificates && \
    mkdir -p /curl-libs

RUN cp /usr/bin/curl /curl-libs/ && \
    ldd /usr/bin/curl | grep "=>" | awk '{print $3}' | xargs -I '{}' cp -v '{}' /curl-libs/

FROM gcr.io/distroless/base

COPY --from=builder /app/bot /
COPY --from=curl-stage /curl-libs/ /usr/lib/
COPY --from=curl-stage /curl-libs/curl /usr/bin/curl
COPY config /config

ENV METRICS_PORT=9090
HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD /usr/bin/curl -f http://localhost:${METRICS_PORT}/health

CMD ["/bot"]
