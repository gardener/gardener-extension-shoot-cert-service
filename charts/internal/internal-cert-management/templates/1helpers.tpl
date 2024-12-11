{{/* HINT: This file is intentionally NOT called _helpers.tpl (as usual) since this Helm chart is embedded via go embed. */}}
{{/* HINT: go embed does not support hidden files, hence, _helpers.tpl cannot be used as name. */}}

{{/* vim: set filetype=mustache: */}}

{{/*
Get a uniq cluster role name.
*/}}
{{- define "clusterRoleName" -}}
{{- if .Values.internalDeployment -}}
extensions.gardener.cloud:extension-shoot-cert-service:{{ .Values.certClass }}
{{- else -}}
extensions.gardener.cloud:extension-shoot-cert-service:shoot
{{- end -}}
{{- end -}}