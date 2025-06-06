{{- define "name" -}}
{{- if .Values.gardener.runtimeCluster.enabled -}}
gardener-extension-shoot-cert-service-runtime
{{- else -}}
gardener-extension-shoot-cert-service
{{- end -}}
{{- end -}}

{{- define "certconfig" -}}
---
apiVersion: shoot-cert-service.extensions.config.gardener.cloud/v1alpha1
kind: Configuration
issuerName: {{ required ".Values.certificateConfig.defaultIssuer.name is required" .Values.certificateConfig.defaultIssuer.name }}
restrictIssuer: {{ required ".Values.certificateConfig.defaultIssuer.restricted is required" .Values.certificateConfig.defaultIssuer.restricted }}
{{- if .Values.certificateConfig.defaultRequestsPerDayQuota }}
defaultRequestsPerDayQuota: {{ .Values.certificateConfig.defaultRequestsPerDayQuota }}
{{- end }}
{{- if .Values.certificateConfig.shootIssuers }}
shootIssuers:
  enabled: {{ .Values.certificateConfig.shootIssuers.enabled }}
{{- end }}
{{- if .Values.certificateConfig.privateKeyDefaults }}
privateKeyDefaults:
{{- if .Values.certificateConfig.privateKeyDefaults.algorithm }}
  algorithm: {{ .Values.certificateConfig.privateKeyDefaults.algorithm }}
{{- end }}
{{- if .Values.certificateConfig.privateKeyDefaults.sizeRSA }}
  sizeRSA: {{ .Values.certificateConfig.privateKeyDefaults.sizeRSA }}
{{- end }}
{{- if .Values.certificateConfig.privateKeyDefaults.sizeECDSA }}
  sizeECDSA: {{ .Values.certificateConfig.privateKeyDefaults.sizeECDSA }}
{{- end }}
{{- end }}
{{- if .Values.certificateConfig.inClusterACMEServerNamespaceMatchLabel }}
inClusterACMEServerNamespaceMatchLabel:
{{ toYaml .Values.certificateConfig.inClusterACMEServerNamespaceMatchLabel | indent 2 }}
{{- end }}
{{- if .Values.certificateConfig.defaultIssuer.acme }}
acme:
  email: {{ required ".Values.certificateConfig.defaultIssuer.acme.email is required" .Values.certificateConfig.defaultIssuer.acme.email }}
  server: {{ required ".Values.certificateConfig.defaultIssuer.acme.server is required" .Values.certificateConfig.defaultIssuer.acme.server }}
  {{- if .Values.certificateConfig.defaultIssuer.acme.propagationTimeout }}
  propagationTimeout: {{ .Values.certificateConfig.defaultIssuer.acme.propagationTimeout }}
  {{- end }}
  {{- if .Values.certificateConfig.defaultIssuer.acme.privateKey }}
  privateKey: |
{{ .Values.certificateConfig.defaultIssuer.acme.privateKey | trim | indent 4 }}
  {{- end }}
  {{- if .Values.certificateConfig.precheckNameservers }}
  precheckNameservers: {{ .Values.certificateConfig.precheckNameservers }}
  {{- end }}
  {{- if .Values.certificateConfig.caCertificates }}
  caCertificates: {{- toYaml .Values.certificateConfig.caCertificates | indent 2 }}
  {{- end }}
  {{- if .Values.certificateConfig.deactivateAuthorizations }}
  deactivateAuthorizations: true
  {{- end }}
  {{- if .Values.certificateConfig.defaultIssuer.acme.skipDNSChallengeValidation }}
  skipDNSChallengeValidation: true
  {{- end }}
{{- end }}
{{- if .Values.certificateConfig.defaultIssuer.ca }}
ca:
  certificate: {{- toYaml (required ".Values.certificateConfig.defaultIssuer.ca.certificate is required" .Values.certificateConfig.defaultIssuer.ca.certificate) | indent 2 }}
  certificateKey: {{- toYaml (required ".Values.certificateConfig.defaultIssuer.ca.certificateKey is required" .Values.certificateConfig.defaultIssuer.ca.certificateKey) | indent 2 }}
{{- end }}
{{- end }}

{{-  define "image" -}}
  {{- if .Values.skaffoldImage }}
  {{- .Values.skaffoldImage }}
  {{- else }}
    {{- if hasPrefix "sha256:" .Values.image.tag }}
    {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
    {{- else }}
    {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
    {{- end }}
  {{- end }}
{{- end }}

{{- define "leaderelectionid" -}}
extension-shoot-cert-service-leader-election
{{- end -}}