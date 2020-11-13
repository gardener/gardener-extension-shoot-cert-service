############# builder
FROM golang:1.15.5 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-cert-service
COPY . .
RUN make install

############# gardener-extension-shoot-cert-service
FROM alpine:3.12.0 AS gardener-extension-shoot-cert-service

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-cert-service /gardener-extension-shoot-cert-service
ENTRYPOINT ["/gardener-extension-shoot-cert-service"]
