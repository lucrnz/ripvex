FROM debian:trixie-slim AS curl-version
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get install -y --no-install-recommends curl jq && \
  rm -rf /var/lib/apt/lists/* && \
  curl -s https://api.github.com/repos/curl/curl/releases/latest | \
  jq -r '.tag_name' | \
  sed 's/^curl-//' | \
  sed 's/_/./g' > /curl-version.txt

FROM golang:1.25.5-trixie AS builder
ARG VERSION_PREFIX=dev
ARG VERSION_DATE
ENV GOOS=linux
WORKDIR /app
COPY . .

COPY --from=curl-version /curl-version.txt /curl-version.txt
RUN CURL_VERSION="$(cat /curl-version.txt)" && \
  rm -f /curl-version.txt && \
  apt-get update && \
  apt-get install -y --no-install-recommends make && \
  rm -rf /var/lib/apt/lists/* && \
  make build VERSION_PREFIX=$(VERSION_PREFIX) VERSION_DATE=$(VERSION_DATE) CURL_VERSION=$(CURL_VERSION)

FROM debian:trixie-slim AS certs
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/*

FROM scratch
COPY --from=builder /app/build/ripvex /ripvex
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/ripvex"]
