# [Gardener Extension for certificate services](https://gardener.cloud)

[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardener-extension-shoot-cert-service)](https://goreportcard.com/report/github.com/gardener/gardener-extension-shoot-cert-service)

Project Gardener implements the automated management and operation of [Kubernetes](https://kubernetes.io/) clusters as a service. Its main principle is to leverage Kubernetes concepts for all of its tasks.

Recently, most of the vendor specific logic has been developed [in-tree](https://github.com/gardener/gardener). However, the project has grown to a size where it is very hard to extend, maintain, and test. With [GEP-1](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md) we have proposed how the architecture can be changed in a way to support external controllers that contain their very own vendor specifics. This way, we can keep Gardener core clean and independent.

## Configuration

Example configuration for this extension controller:

```yaml
apiVersion: shoot-cert-service.extensions.config.gardener.cloud/v1alpha1
kind: Configuration
issuerName: gardener
acme:
  email: john.doe@example.com
  server: https://acme-v02.api.letsencrypt.org/directory
# privateKey: | # Optional key for Let's Encrypt account.
#   -----BEGIN BEGIN RSA PRIVATE KEY-----
#   ...
#   -----END RSA PRIVATE KEY-----
```

## Extension-Resources

Example extension resource:

```yaml
apiVersion: extensions.gardener.cloud/v1alpha1
kind: Extension
metadata:
  name: "extension-certificate-service"
  namespace: shoot--project--abc
spec:
  type: shoot-cert-service
```

When an extension resource is reconciled, the extension controller will create an instance of [Cert-Management](https://github.com/gardener/cert-management) as well as an `Issuer` with the ACME information provided in the [configuration](#Configuration) above. These resources are placed inside the shoot namespace on the seed. Also, the controller takes care about generating necessary `RBAC` resources for the seed as well as for the shoot.

Please note, this extension controller relies on the [Gardener-Resource-Manager](https://github.com/gardener/gardener-resource-manager) to deploy k8s resources to seed and shoot clusters, i.e. it never deploys them directly.

## Feedback and Support

Feedback and contributions are always welcome. Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extension-shoot-cert-service/issues) or join our [Slack channel #gardener](https://kubernetes.slack.com/messages/gardener) (please invite yourself to the Kubernetes workspace [here](http://slack.k8s.io)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)
