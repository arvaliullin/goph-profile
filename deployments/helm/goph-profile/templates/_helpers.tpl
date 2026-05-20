{{- define "goph-profile.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "goph-profile.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "goph-profile.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "goph-profile.labels" -}}
helm.sh/chart: {{ include "goph-profile.chart" . }}
{{ include "goph-profile.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "goph-profile.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goph-profile.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "goph-profile.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "goph-profile.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "goph-profile.componentName" -}}
{{- printf "%s-%s" (include "goph-profile.fullname" .ctx) .name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "goph-profile.componentLabels" -}}
{{ include "goph-profile.labels" .ctx }}
app.kubernetes.io/component: {{ .name }}
{{- end -}}

{{- define "goph-profile.componentSelectorLabels" -}}
{{ include "goph-profile.selectorLabels" .ctx }}
app.kubernetes.io/component: {{ .name }}
{{- end -}}

{{- define "goph-profile.waitForDepsInitContainers" }}
{{- if .Values.waitForDeps.enabled }}
initContainers:
  - name: wait-for-deps
    image: {{ .Values.waitForDeps.image | quote }}
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
      runAsNonRoot: false
      runAsUser: 0
      seccompProfile:
        type: RuntimeDefault
    resources:
      limits:
        cpu: 50m
        memory: 32Mi
      requests:
        cpu: 10m
        memory: 16Mi
    command:
      - sh
      - -c
      - |
        set -eu
        HOST="{{ .Values.waitForDeps.host }}"
{{- range .Values.waitForDeps.ports }}
        until nc -z "$HOST" {{ . }}; do echo "waiting ${HOST}:{{ . }}"; sleep 2; done
{{- end }}
        echo "deps reachable"
{{- end }}
{{- end }}
