FROM golang:1.23.1 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
COPY .env /app/.env

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot ./cmd

FROM ubuntu:22.04 as curl-stage

RUN apt-get clean && \
    apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /curl-libs

RUN cp /usr/bin/curl /curl-libs/
RUN ldd /usr/bin/curl | grep "=>" | awk '{print $3}' | xargs -I '{}' cp -v '{}' /curl-libs/

FROM gcr.io/distroless/base

COPY --from=builder /app/bot /
COPY --from=curl-stage /curl-libs/ /usr/lib/
COPY --from=curl-stage /curl-libs/curl /usr/bin/curl
COPY config /config

CMD ["/bot"]
