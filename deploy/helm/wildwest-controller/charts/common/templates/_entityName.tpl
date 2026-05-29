{{- define "common.entity.name" -}}
{{- if contains .Chart.Name .Release.Name }}
{{- printf "%s" .Chart.Name | trunc 63 }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 }}
{{- end -}}
{{- end -}}

{{- define "release.name" -}}
{{- printf "%s" .Release.Name | trunc 63 }}
{{- end -}}
