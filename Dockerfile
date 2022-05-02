############# builder
FROM eu.gcr.io/gardener-project/3rd/golang:1.17.9 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-cert-service
COPY . .
RUN make install

############# gardener-extension-shoot-cert-service
FROM alpine:3.15.3 AS gardener-extension-shoot-cert-service

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-cert-service /gardener-extension-shoot-cert-service
ENTRYPOINT ["/gardener-extension-shoot-cert-service"]
