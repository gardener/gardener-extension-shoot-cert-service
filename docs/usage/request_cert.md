# Request X.509 Certificates 

## Introduction
Dealing with applications on Kubernetes which offer service endpoints (e.g. HTTP) may also require you to enable a 
secured communication via SSL/TLS. Gardener let's you request a commonly trusted X.509 certificate for your application 
endpoint. Furthermore, Gardener takes care about the renewal process for your requested certificate.

Let's get the basics straight first. If this is too long for you, you can read below how to get certificates by

 - [Certificate Resources](#request-a-certificate-via-certificate)
 - [Ingress](#request-a-certificate-via-ingress)
 - [Service](#request-a-certificate-via-service) 


## Restrictions

### Service Scope
This service enables users to request managed X.509 certificates with the help of [ACME](https://tools.ietf.org/html/rfc8555) and [Let's Encrypt](https://letsencrypt.org/).
It __does not__ equip or manage DNS records for cluster assets like `Services` or `Ingresses`. Thus, you can obtain a valid certificate but your service might still not be resolvable or reachable due to missing DNS configuration. Please consult [this page](https://github.com/gardener/gardener-extension-shoot-dns-service/tree/master/docs/usage/dns_names.md) if your services require managed DNS records.

### Supported Domains
Certificates may be obtained for any subdomain of your shoot's domain (see `.spec.dns.domain` of your shoot resource) with the default `issuer`. For custom domains, please consult [custom domains](#Custom-Domains).

### Character Restrictions
Due to the ACME protocol specification, at least one domain of the domains you request a certificate for must not exceed a character limit of 64  (CN - Common Name).

For example, the following request is invalid:

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
metadata:
  name: cert-invalid
  namespace: default
spec:
  commonName: morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
```

But it is valid to request a certificate for this domain if you have at least one domain which does not exceed the mentioned limit:

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
metadata:
  name: cert-example
  namespace: default
spec:
  commonName: short.ingress.shoot.project.default-domain.gardener.cloud
  dnsNames:
  - morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
```

## Certificate Resources
Every X.509 certificate is represented by a Kubernetes custom resource `certificate.cert.gardener.cloud` in your cluster. A `Certificate` resource may be used to initiate a new certificate request as well as to manage its lifecycle. Gardener's certificate service regularly checks the expiration timestamp of Certificates, triggers a renewal process if necessary and replaces the existing X.509 certificate with a new one.

> Your application should be able to reload replaced certificates in a timely manner to avoid service disruptions.

Certificates can either be requested by creating `Certificate` resources in the Kubernetes cluster or by configuring `Ingress` or `Service` (type `LoadBalancer`) resources. If the latter is used, a `Certificate` resource will automatically be created by Gardener's certificate service.

If you're interested in the current progress of your request, you're advised to consult the `Certificate`'s `status` subresource. You'll also find error descriptions in the `status` in case the issuance failed.

Certificate status example:

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
...
status:
  commonName: short.ingress.shoot.project.default-domain.gardener.cloud
  expirationDate: "2020-02-27T15:39:10Z"
  issuerRef:
    name: garden
    namespace: shoot--foo--bar
  lastPendingTimestamp: "2019-11-29T16:38:40Z"
  observedGeneration: 11
  state: Ready
```

## Custom Domains
If you want to request certificates for domains other then any subdomain of `shoot.spec.dns.domain`, the following configuration is required:

### DNS provider
In order to issue certificates for a custom domain you need to specify a DNS provider which is permitted to create DNS records for subdomains of your requested domain in the certificate. For example, if you request a certificate for `host.example.com` your DNS provider must be capable of managing subdomains of `host.example.com`.

DNS providers are normally specified in the shoot manifest. 

If the `DNSProvider` replication feature is enabled, an provider can alternatively defined in
the shoot cluster.

#### Provider in the shoot manifest

Example for a provider in the shoot manifest:

```yaml
kind: Shoot
...
spec:
  dns:
    providers:
    - type: aws-route53 # consult the DNS provisioning controllers group (dnscontrollers) in https://github.com/gardener/external-dns-management#using-the-dns-controller-manager for possible values
      secretName: provider-example-com # contains credentials for service account, see any 20-secret-<provider>-credentials.yaml in https://github.com/gardener/external-dns-management/tree/master/examples
```

The secret referenced by `secretName` can also be conveniently created via the Gardener dashboard.

#### Provider resouce in the shoot cluster

*Prerequiste*: The `DNSProvider` replication feature has to be enabled.
It is either enabled globally in the `ControllerDeployment` or in the shoot manifest
with:

```yaml
...
spec:
  extensions:
    - type: shoot-dns-service
      providerConfig:
        apiVersion: service.dns.extensions.gardener.cloud/v1alpha1
        kind: DNSConfig
        dnsProviderReplication:
          enabled: true
...
```

Example for specifying a `DNSProvider` resource and its `Secret` in any namespace of the shoot cluster:

```yaml
apiVersion: dns.gardener.cloud/v1alpha1
kind: DNSProvider
metadata:
  annotations:
    dns.gardener.cloud/class: garden  
  name: my-own-domain
  namespace: my-namespace
spec:
  type: aws-route53
  secretRef:
    name: my-own-domain-credentials
  domains:
    include:
    - my.own.domain.com
---
apiVersion: v1
kind: Secret
metadata:
  name: my-own-domain-credentials
  namespace: my-namespace
type: Opaque
data:
  # replace '...' with values encoded as base64
  AWS_ACCESS_KEY_ID: ...
  AWS_SECRET_ACCESS_KEY: ...
```

### Issuer
Another prerequisite to request certificates for custom domains is a dedicated issuer.

Note: This is only needed if the default issuer provided by Gardener is restricted to shoot related domains or you are using
domain names not visible to public DNS servers. You may therefore try first without defining an own issuer.

The custom issuers are specified normally in the shoot manifest.

If the `shootIssuers` feature is enabled, it can alternatively be defined in the shoot cluster.

#### Issuer in the shoot manifest

Example for an issuer in the shoot manifest:

```yaml
kind: Shoot
...
spec:
  extensions:
  - type: shoot-cert-service
    providerConfig:
      apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
      kind: CertConfig
      issuers:
        - email: your-email@example.com
          name: custom-issuer # issuer name must be specified in every custom issuer request, must not be "garden"
          server: 'https://acme-v02.api.letsencrypt.org/directory'
          privateKeySecretName: my-privatekey # referenced resource, the private key must be stored in the secret at `data.privateKey`
      #shootIssuers:
      #  enabled: true # if true, allows to specify issuers in the shoot cluster

      #precheckNameservers: "10.0.0.53,10.123.56.53,8.8.8.8" # optional comma separated list of DNS server IP addresses if public DNS servers are not sufficient for prechecking DNS challenges

  resources:
  - name: my-privatekey
    resourceRef:
      apiVersion: v1
      kind: Secret
      name: custom-issuer-privatekey # name of secret in Gardener project
```

If you are using an ACME provider for private domains, you may need to change the nameservers used for
checking the availability of the DNS challenge's TXT record before the certificate is requested from the ACME provider.
By default, only public DNS servers may be used for this purpose.
At least one of the `precheckNameservers` must be able to resolve the private domain names. 

####

*Prerequiste*: The `shootIssuers` feature has to be enabled.
It is either enabled globally in the `ControllerDeployment` or in the shoot manifest
with:

```yaml
kind: Shoot
...
spec:
  extensions:
  - type: shoot-cert-service
    providerConfig:
      apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
      kind: CertConfig
      shootIssuers:
        enabled: true # if true, allows to specify issuers in the shoot cluster
...
```

Example for specifying an `Issuer` resource and its `Secret` directly in any
namespace of the shoot cluster:

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Issuer
metadata:
  name: my-own-issuer
  namespace: my-namespace
spec:
  acme:
    domains:
      include:
      - my.own.domain.com
    email: some.user@my.own.domain.com
    privateKeySecretRef:
      name: my-own-issuer-secret
      namespace: my-namespace
    server: https://acme-v02.api.letsencrypt.org/directory
---
apiVersion: v1
kind: Secret
metadata:
  name: my-own-issuer-secret
  namespace: my-namespace
type: Opaque
data:
  privateKey: ... # replace '...' with valus encoded as base64
```

## Examples
### Request a certificate via Certificate

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
metadata:
  name: cert-example
  namespace: default
spec:
  commonName: short.ingress.shoot.project.default-domain.gardener.cloud
  dnsNames:
  - morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
  secretRef:
    name: cert-example
    namespace: default
# issuerRef:
#   name: custom-issuer
```

|  Path  |  Description  |
|:----|:----|
| `spec.commonName` (required) |  Specifies for which domain the certificate request will be created. This entry must comply with the [64 character](#Character-Restrictions) limit.  |
| `spec.dnsName` |  Additional domains the certificate should be valid for. Entries in this list can be longer than 64 characters.  |
| `spec.secretRef` |  Specifies the secret which contains the certificate/key pair. If the secret is not available yet, it'll be created automatically as soon as the X.509 certificate has been issued.  |
| `spec.issuerRef` |  Specifies the issuer you want to use. Only necessary if you request certificates for [custom domains](#Custom-Domains).  |

### Request a wildcard certificate via Certificate

```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
metadata:
  name: cert-wildcard
  namespace: default
spec:
  commonName: '*.ingress.shoot.project.default-domain.gardener.cloud'
  secretRef:
    name: cert-wildcard
    namespace: default
# issuerRef:
#   name: custom-issuer
```

|  Path  |  Description  |
|:----|:----|
| `spec.commonName` (required) |  Specifies for which domain the certificate request will be created. This entry must comply with the [64 character](#Character-Restrictions) limit.  |
| `spec.secretRef` |  Specifies the secret which contains the certificate/key pair. If the secret is not available yet, it'll be created automatically as soon as the X.509 certificate has been issued.  |
| `spec.issuerRef` |  Specifies the issuer you want to use. Only necessary if you request certificates for [custom domains](#Custom-Domains).  |

### Request a certificate via Ingress

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: vuejs-ingress
  annotations:
    cert.gardener.cloud/purpose: managed
  # cert.gardener.cloud/issuer: custom-issuer
spec:
  tls:
  # Must not exceed 64 characters.
  - hosts:
    - short.ingress.shoot.project.default-domain.gardener.cloud
    - morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
    # Certificate and private key reside in this secret.
    secretName: testsecret-tls
  rules:
  - host: morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
    http:
      paths:
      - backend:
          serviceName: vuejs-svc
          servicePort: 8080
```

|  Path  |  Description  |
|:----|:----|
| `metadata.annotations` |  Annotations should have `cert.gardener.cloud/purpose: managed` to activate the certificate service on this resource. `cert.gardener.cloud/issuer: <name>` is optional and may be specified if the certificate is request for a [custom domains](#Custom-Domains).  |
| `spec.tls[].hosts` |  Specifies for which domains the certificate request will be created. The first entry is always taken to fill the `Common Name` field and must therefore comply with the [64 character](#Character-Restrictions) limit.  |
| `spec.tls[].secretName` | Specifies the secret which contains the certificate/key pair to be used by this Ingress. If the secret is not available yet, it'll be created automatically as soon as the certificate has been issued. Once configured, you're not advised to change the name while the Ingress is still managed by the certificate service.  |

### Request a wildcard certificate via Ingress

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: vuejs-ingress
  annotations:
    cert.gardener.cloud/purpose: managed
  # cert.gardener.cloud/issuer: custom-issuer
spec:
  tls:
  # Must not exceed 64 characters.
  - hosts:
    - "*.ingress.shoot.project.default-domain.gardener.cloud"
    # Certificate and private key reside in this secret.
    secretName: testsecret-tls
  rules:
  - host: morethan64characters.ingress.shoot.project.default-domain.gardener.cloud
    http:
      paths:
      - backend:
          serviceName: vuejs-svc
          servicePort: 8080
```

> Domains must not overlap when requesting a wildcard certificate. For example, requests for `*.example.com` must not contain `foo.example.com` at the same time.

|  Path  |  Description  |
|:----|:----|
| `metadata.annotations` |  Annotations should have `cert.gardener.cloud/purpose: managed` to activate the certificate service on this resource. `cert.gardener.cloud/issuer: <name>` is optional and may be specified if the certificate is request for a [custom domains](#Custom-Domains).  |
| `spec.tls[].hosts` |  Specifies for which domains the certificate request will be created. The first entry is always taken to fill the `Common Name` field and must therefore comply with the [64 character](#Character-Restrictions) limit.  |
| `spec.tls[].secretName` | Specifies the secret which contains the certificate/key pair to be used by this Ingress. If the secret is not available yet, it'll be created automatically as soon as the certificate has been issued. Once configured, you're not advised to change the name while the Ingress is still managed by the certificate service.  |

### Request a certificate via Service

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    cert.gardener.cloud/secretname: test-service-secret
  # cert.gardener.cloud/issuer: custom-issuer
    dns.gardener.cloud/dnsnames: "service.shoot.project.default-domain.gardener.cloud, morethan64characters.svc.shoot.project.default-domain.gardener.cloud"
    dns.gardener.cloud/ttl: "600"
  name: test-service
  namespace: default
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 8080
  type: LoadBalancer
```

|  Path  |  Description  |
|:----|:----|
| `metadata.annotations[cert.gardener.cloud/secretname]` |  Specifies the secret which contains the certificate/key pair. If the secret is not available yet, it'll be created automatically as soon as the certificate has been issued.  |
| `metadata.annotations[cert.gardener.cloud/issuer]` |  Optional and may be specified if the certificate is request for a [custom domains](#Custom-Domains).  |
| `metadata.annotations[dns.gardener.cloud/dnsnames]` | Specifies for which domains the certificate request will be created. The first entry is always taken to fill the `Common Name` field and must therefore comply with the [64 character](#Character-Restrictions) limit.  |

### Request a wildcard certificate via Service

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    cert.gardener.cloud/secretname: test-service-secret
  # cert.gardener.cloud/issuer: custom-issuer
    dns.gardener.cloud/dnsnames: "*.service.shoot.project.default-domain.gardener.cloud"
    dns.gardener.cloud/ttl: "600"
  name: test-service
  namespace: default
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 8080
  type: LoadBalancer
```

> Domains must not overlap when requesting a wildcard certificate. For example, requests for `*.example.com` must not contain `foo.example.com` at the same time.

|  Path  |  Description  |
|:----|:----|
| `metadata.annotations[cert.gardener.cloud/secretname]` |  Specifies the secret which contains the certificate/key pair. If the secret is not available yet, it'll be created automatically as soon as the certificate has been issued.  |
| `metadata.annotations[cert.gardener.cloud/issuer]` |  Optional and may be specified if the certificate is request for a [custom domains](#Custom-Domains).  |
| `metadata.annotations[dns.gardener.cloud/dnsnames]` | Specifies for which domains the certificate request will be created. The first entry is always taken to fill the `Common Name` field and must therefore comply with the [64 character](#Character-Restrictions) limit.  |

## Quotas

For security reasons there may be a default quota on the certificate requests per day set globally in the controller
registration of the shoot-cert-service. 

The default quota only applies if there is no explicit quota defined for the issuer itself with the field
`requestsPerDayQuota`, e.g.:

```yaml
kind: Shoot
...
spec:
  extensions:
  - type: shoot-cert-service
    providerConfig:
      apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
      kind: CertConfig
      issuers:
        - email: your-email@example.com
          name: custom-issuer # issuer name must be specified in every custom issuer request, must not be "garden"
          server: 'https://acme-v02.api.letsencrypt.org/directory'
          requestsPerDayQuota: 10
```
