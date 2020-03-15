############# builder
FROM golang:1.13.8 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-cert-service
COPY . .
RUN make install-requirements && make VERIFY=true all

############# gardener-extension-shoot-cert-service
FROM alpine:3.11.3 AS gardener-extension-shoot-cert-service

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-cert-service /gardener-extension-shoot-cert-service
ENTRYPOINT ["/gardener-extension-shoot-cert-service"]
