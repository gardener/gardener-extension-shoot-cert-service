############# builder
FROM golang:1.21.0 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-cert-service
COPY . .
RUN make install

############# gardener-extension-shoot-cert-service
FROM gcr.io/distroless/static-debian11:nonroot AS gardener-extension-shoot-cert-service
WORKDIR /

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-shoot-cert-service /gardener-extension-shoot-cert-service
ENTRYPOINT ["/gardener-extension-shoot-cert-service"]
