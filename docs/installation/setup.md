# Gardener Certificate Management

## Introduction
Gardener comes with an extension that enables shoot owners to request X.509 compliant certificates for shoot domains.

## Extension Installation
The `Shoot-Cert-Service` extension can be deployed and configured via Gardener's native resource [ControllerRegistration](https://github.com/gardener/gardener/blob/master/docs/extensions/controllerregistration.md).

### Prerequisites
To let the `Shoot-Cert-Service` operate properly, you need to have:
- a [DNS service](https://github.com/gardener/external-dns-management) in your seed
- contact details and optionally a private key for a pre-existing [Let's Encrypt](https://letsencrypt.org/) account

### ControllerRegistration
An example of a `ControllerRegistration` for the `Shoot-Cert-Service` can be found here: https://github.com/gardener/gardener-extension-shoot-cert-service/blob/master/example/controller-registration.yaml

The `ControllerRegistration` contains a Helm chart which eventually deploy the `Shoot-Cert-Service` to seed clusters. It offers some configuration options, mainly to set up a default issuer for shoot clusters. With a default issuer, pre-existing Let's Encrypt accounts can be used and shared with shoot clusters (See "One Account or Many?" of the [Integration Guide](https://letsencrypt.org/docs/integration-guide/)).

> Please keep the Let's Encrypt [Rate Limits](https://letsencrypt.org/docs/rate-limits/) in mind when using this shared account model. Depending on the amount of shoots and domains it is recommended to use an account with increased rate limits.

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: ControllerRegistration
...
  values:
    certificateConfig:
      defaultIssuer:
        acme:
            email: foo@example.com
            privateKey: |-
            -----BEGIN RSA PRIVATE KEY-----
            ...
            -----END RSA PRIVATE KEY-----
            server: https://acme-v02.api.letsencrypt.org/directory
        name: default-issuer
#       restricted: true # restrict default issuer to any sub-domain of shoot.spec.dns.domain

#     defaultRequestsPerDayQuota: 50

#     precheckNameservers: 8.8.8.8,8.8.4.4

#     caCertificates: | # optional custom CA certificates when using private ACME provider
#     -----BEGIN CERTIFICATE-----
#     ...
#     -----END CERTIFICATE-----
#
#     -----BEGIN CERTIFICATE-----
#     ...
#     -----END CERTIFICATE-----

      shootIssuers:
        enabled: false # if true, allows to specify issuers in the shoot clusters

```

#### Enablement

If the `Shoot-Cert-Service` should be enabled for every shoot cluster in your Gardener managed environment, you need to globally enable it in the `ControllerRegistration`:
```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: ControllerRegistration
...
  resources:
  - globallyEnabled: true
    kind: Extension
    type: shoot-cert-service
```

Alternatively, you're given the option to only enable the service for certain shoots:
```yaml
kind: Shoot
apiVersion: core.gardener.cloud/v1beta1
...
spec:
  extensions:
  - type: shoot-cert-service
...
```
