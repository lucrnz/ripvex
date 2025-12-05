FROM alpine/curl:latest AS curl-version
RUN curl --version | awk -F' ' '{print $2}' > /curl-version.txt

FROM golang:1.25.5-alpine AS builder
ARG VERSION_PREFIX=dev
ARG VERSION_DATE
ENV GOOS=linux
WORKDIR /app
COPY . .

COPY --from=curl-version /curl-version.txt /curl-version.txt
RUN CURL_VERSION="$(cat /curl-version.txt)" && \
  rm -f /curl-version.txt && \
  apk add --no-cache make && \
  make build VERSION_PREFIX=$(VERSION_PREFIX) VERSION_DATE=$(VERSION_DATE) CURL_VERSION=$$CURL_VERSION

FROM alpine:latest AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=builder /app/build/ripvex /ripvex
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/ripvex"]
