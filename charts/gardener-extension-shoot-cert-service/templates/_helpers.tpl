{{- define "name" -}}
gardener-extension-shoot-cert-service
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
{{- end }}

{{-  define "image" -}}
  {{- if hasPrefix "sha256:" .Values.image.tag }}
  {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
  {{- else }}
  {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
  {{- end }}
{{- end }}

{{- define "priorityclassversion" -}}
{{- if semverCompare ">= 1.14-0" .Capabilities.KubeVersion.GitVersion -}}
scheduling.k8s.io/v1
{{- else if semverCompare ">= 1.11-0" .Capabilities.KubeVersion.GitVersion -}}
scheduling.k8s.io/v1beta1
{{- else -}}
scheduling.k8s.io/v1alpha1
{{- end -}}
{{- end -}}

{{- define "leaderelectionid" -}}
extension-shoot-cert-service-leader-election
{{- end -}}