---
title: Changing alerting settings
level: beginner
category: Networking
scope: operator
publishdate: 2023-07-20
tags: ["task"]
description: How to change the alerting on expiring certificates
---

# Changing alerting settings

Certificates are normally renewed automatically 30 days before they expire.
As a second defense line, there is an alerting in Prometheus activated if the certificate is a few days
before expiration. By default, the alert is triggered 15 days before expiration.

You can configure the days in the `providerConfig` of the extension.

In this example, the days are changed to 3 days before expiration.

```yaml
kind: Shoot
...
spec:
  extensions:
  - type: shoot-cert-service
    providerConfig:
      apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
      kind: CertConfig
      alerting:
        certExpirationAlertDays: 3
```
