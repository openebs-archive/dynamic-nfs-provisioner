{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "nfspv.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nfspv.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "nfspv.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "nfspv.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "localpv.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Meta labels
*/}}
{{- define "nfspv.common.metaLabels" -}}
chart: {{ include "localpv.chart" . }}
heritage: {{ .Release.Service }}
openebs.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
Selector Labels
*/}}
{{- define "nfspv.selectorLabels" -}}
app: {{ include "nfspv.name" . }}
release: {{ .Release.Name }}
component: {{ .Values.nfspv.name }}
{{- end }}

{{/*
Component labels
*/}}
{{- define "nfspv.componentLabels" -}}
openebs.io/component-name: openebs-{{ .Values.nfspv.name }}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "nfspv.labels" -}}
{{ include "nfspv.common.metaLabels" . }}
{{ include "nfspv.selectorLabels" . }}
{{ include "nfspv.componentLabels" . }}
{{- end -}}
