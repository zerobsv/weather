{{- define "weather.fullname" -}}
{{ printf "%s-service" .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

