############# builder
FROM eu.gcr.io/gardener-project/3rd/golang:1.16.7 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-cert-service
COPY . .
RUN make install

############# gardener-extension-shoot-cert-service
FROM eu.gcr.io/gardener-project/3rd/alpine:3.13.5 AS gardener-extension-shoot-cert-service

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-cert-service /gardener-extension-shoot-cert-service
ENTRYPOINT ["/gardener-extension-shoot-cert-service"]
