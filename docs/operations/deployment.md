# Gardener Certificate Management

## Introduction
Gardener comes with an extension that enables shoot owners to request X.509 compliant certificates for shoot domains.
There are two ways to deploy the `Shoot-Cert-Service` extension:
- the new way via Gardener's resource [Extension (Operator)](https://github.com/gardener/gardener/blob/master/docs/extensions/operator-extension.md) (recommended)
- the traditional way via Gardener's resource [ControllerRegistration](https://github.com/gardener/gardener/blob/master/docs/extensions/controllerregistration.md)

## Extension Installation via Operator `Extension`
The `Shoot-Cert-Service` extension can be deployed and configured via Gardener's native resource [Extension (Operator)](https://github.com/gardener/gardener/blob/master/docs/extensions/operator-extension.md).
Here, the Gardener Operator controls the deployment of `Shoot-Cert-Service`:
- It ensures that the `ControllerRegistration` and `ControllerDeployment` is created in the virtual garden
- Optionally it deploys the `cert-controller-manager` to the Garden Runtime cluster. Additionally it may automatically 
  create the TLS certificate for the virtual kube-apiserver and ingress on the runtime cluster.
- Optionally it deploys the `cert-controller-manager` to the seed and automatically creates the TLS certificates for ingress on the seed itself.

### Prerequisites
To let the `Shoot-Cert-Service` operate properly, you need to have:
- a [DNS service](https://github.com/gardener/external-dns-management) in your seed
- contact details and optionally a private key for a pre-existing [Let's Encrypt](https://letsencrypt.org/) account (or other backend supporting ACME)
- alternatively to ACME, a custom CA certificate and key can be used

For the special `cert-controller-manager` deployments on the Garden Runtime and Seed clusters, `DNSRecord` resources
are used to create the necessary DNS records for the ACME challenge. Therefore, the provider extension must also be
installed using an operator `Extension`. The provider extension must support a deployment on the Garden Runtime cluster.

### Optional deployment of the `cert-controller-manager` to the Garden Runtime cluster

If the following `extension.extensions.gardener.cloud` resource is created on the runtime cluster, the Gardener Operator 
will deploy the extension. As soon as the extension is up, it will deploy a `cert-controller-manager` into the `garden` 
namespace working on certificates with the annotation `cert.gardener.cloud/class=garden`.

```yaml
apiVersion: extensions.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: shoot-cert-service
  namespace: garden
spec:
  class: garden
  type: shoot-cert-service
```

Note, that the default issuer for such certificates is defined in the `extension.operator.gardener.cloud` under
`.spec.deployment.extension.runtimeClusterValues.certificateConfig`.

You may now use `Certificate` resources on the runtime cluster.
Example:
```yaml
apiVersion: cert.gardener.cloud/v1alpha1
kind: Certificate
metadata:
  annotations:
    cert.gardener.cloud/class: garden
  name: my-cert
  namespace: my-namespace
spec:
  dnsNames:
  - '*.example.com'
  secretRef:
    name: my-secretname
    namespace: my-namespace
```

Optionally, the management of the garden runtime certificate can be enabled in the `extension.operator.gardener.cloud` resource:

```yaml
apiVersion: operator.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: shoot-cert-service
spec:
  deployment:
    extension:
      runtimeClusterValues:
        gardenerCertificates:
          runtimeCluster:
            enabled: true
          # virtualKubeAPIServerIncludePrimaryDomain: false # set to true to include the first domain of the virtual cluster kube-apiserver
        certificateConfig:
          defaultIssuer:
          # acme: ... # either if you want to use ACME, e.g. Let's Encrypt
          # ca: ...   # or if you want to use a custom CA
```

In this case, the extension will perform additional steps to create a wildcard certificate for the virtual-garden-kube-apiserver service and ingress on the runtime cluster (e.g. for monitoring components).
1. It will run an additional `gardener` controller to fetch the domain names from the `Garden` resource.  
   For the virtual-garden-kube-apiserver from `.spec.virtualCluster.dns.domains` and for the ingress from `.spec.virtualCluster.ingress.domain`.
2. It will create a `Certificate` resource for the wildcard subdomains `*.` for these collected names
3. The cert-controller-manager will request/manage the `Certificate` and create/update the secret `garden/tls`
4. It will run an additional `certificate` controller to watch for this certificate to become ready and then annotates the `virtual-garden-kube-apiserver` deployment.
5. The webhook `sniconfig` will patch the `virtual-garden-kube-apiserver` deployment to use the secret `garden/tls` via `--tls-sni-cert-key` command line option. 

### Optional deployment of `cert-controller-manager` to the Seed cluster

The management of a garden certificate for the seed's control planes can be enabled with this configuration:

```yaml
apiVersion: operator.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: shoot-cert-service
spec:
  deployment:
    extension:
      values:
        gardenerCertificates:
          seed:
            enabled: true
        certificateConfig:
          defaultIssuer:
            # acme: ... # either if you want to use ACME, e.g. Let's Encrypt
            # ca: ...   # or if you want to use a custom CA
      policy: Always # policy should be set to 'Always' to ensure the extension is deployed on all seeds 
```

The extension will be deployed with an `extension.extensions.gardener.cloud` resource in its deployment namespace.
After the extension is up, it will deploy an own `cert-controller-manager`.
This controller is responsible for certificates annotated with `cert.gardener.cloud/class=seed`.
Additionally, it will create a `Certificate` resource named `garden/ingress-wildcard-cert` for the wildcard subdomain `*.` 
of the domain name as specified in the `Seed` resource at `.spec.ingress.domain`.

After the `cert-controller-manager` has reconciled the certificate successfully, it will create or update the
secret `garden/ingress-wildcard-cert` with the label `gardener.cloud/role=controlplane-cert`. 
Later, the Gardenlet may look up the secret by the label and forward it to several control plane components (like kube-apiserver and monitoring).

### Extension

An example of an `Extension` for the `Shoot-Cert-Service` can be found at [extension.operator.yaml](../../example/extension.operator.yaml).

```yaml
apiVersion: operator.gardener.cloud/v1alpha1
kind: Extension
...
spec:
  values:
  # gardenerCertificates:
  #   seed:
  #     enabled: true
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
      
      # alternatively to the 'acme' section, use a custom CA
      # ca:
      #   certificate: |
      #   -----BEGIN CERTIFICATE-----
      #   ...
      #   -----END CERTIFICATE-----
      #   certificateKey: |
      #   -----BEGIN PRIVATE KEY-----
      #   ...
      #   -----END PRIVATE KEY-----
      #   caCertificates: | # optional custom CA certificates when using intermediate CAs
      #   -----BEGIN CERTIFICATE-----
      #   ...
      #   -----END CERTIFICATE-----
      #
      #   -----BEGIN CERTIFICATE-----
      #   ...
      #   -----END CERTIFICATE-----

      shootIssuers:
        enabled: false # if true, allows to specify issuers in the shoot clusters

  runtimeClusterValues:
  # gardenerCertificates:
  #   runtimeCluster:
  #     enabled: true
  #     virtualKubeAPIServerIncludePrimaryDomain: false # set to true to include the first domain of the virtual cluster kube-apiserver
    certificateConfig:
      defaultIssuer:
        # duplicate the issuer configuration from the 'values.certificateConfig.defaultIssuer' section here
  helm:
    ociRepository:
      ref: europe-docker.pkg.dev/gardener-project/releases/charts/gardener/extensions/shoot-cert-service:1.48.0
  policy: Always
```

## Extension Installation via `ControllerRegistration`
The `Shoot-Cert-Service` extension can be deployed and configured via Gardener's native resource [ControllerRegistration](https://github.com/gardener/gardener/blob/master/docs/extensions/controllerregistration.md).

### Prerequisites
To let the `Shoot-Cert-Service` operate properly, you need to have:
- a [DNS service](https://github.com/gardener/external-dns-management) in your seed
- contact details and optionally a private key for a pre-existing [Let's Encrypt](https://letsencrypt.org/) account (or other backend supporting ACME)
- alternatively to ACME, a custom CA certificate and key can be used

### ControllerRegistration
An example of a `ControllerRegistration` for the `Shoot-Cert-Service` can be found at [controller-registration.yaml](../../example/controller-registration.yaml).

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
