{{/* Adapted for TExec from Semaphore UI Charts; see NOTICE and LICENSE. */}}
{{- define "texec.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "texec.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 58 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "texec.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 58 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 58 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- define "texec.selectorLabels" -}}
app.kubernetes.io/name: {{ include "texec.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: server
{{- end -}}
{{- define "texec.labels" -}}
{{ include "texec.selectorLabels" . }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.labels }}
{{ toYaml . }}
{{- end }}
{{- end -}}
{{- define "texec.image" -}}
{{- if .Values.image.digest -}}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository (default (printf "v%s" .Chart.AppVersion) .Values.image.tag) -}}
{{- end -}}
{{- end -}}
{{- define "texec.claimName" -}}
{{- default (include "texec.fullname" .) .Values.persistence.existingClaim -}}
{{- end -}}
{{- define "texec.basePath" -}}
{{- if .Values.general.host -}}
{{- (urlParse .Values.general.host).path | trimSuffix "/" -}}
{{- end -}}
{{- end -}}
{{- define "texec.environment" -}}
TEXEC_INTERFACE: "0.0.0.0"
TEXEC_PORT: "3000"
TEXEC_CONFIG_PATH: "/var/lib/texec/config"
TEXEC_DB_PATH: "/var/lib/texec"
TEXEC_TMP_PATH: "/tmp/texec"
TEXEC_WEB_ROOT: {{ .Values.general.host | quote }}
TEXEC_PASSWORD_LOGIN_DISABLED: {{ .Values.general.passwordLoginDisable | quote }}
TEXEC_NON_ADMIN_CAN_CREATE_PROJECT: {{ .Values.general.nonAdminCanCreateProject | quote }}
TEXEC_FORWARDED_ENV_VARS: {{ .Values.config.forwarded_env_vars | toJson | quote }}
TEXEC_OIDC_PROVIDERS: {{ include "texec.oidcProviders" . | quote }}
TEXEC_ADMIN: {{ .Values.admin.username | quote }}
TEXEC_ADMIN_NAME: {{ .Values.admin.fullname | quote }}
TEXEC_ADMIN_EMAIL: {{ .Values.admin.email | quote }}
TEXEC_DB_DIALECT: {{ .Values.database.type | quote }}
TEXEC_DB_OPTIONS: {{ toJson .Values.database.options | quote }}
{{- if eq .Values.database.type "sqlite" }}
TEXEC_DB_HOST: "/var/lib/texec/database.sqlite"
{{- else }}
TEXEC_DB_HOST: {{ required "database.host is required for an external database" .Values.database.host | quote }}
TEXEC_DB_PORT: {{ default (ternary 5432 3306 (eq .Values.database.type "postgres")) .Values.database.port | quote }}
TEXEC_DB: {{ .Values.database.name | quote }}
{{- end }}
{{- end -}}

{{/* ServiceAccount and provider dictionary follow semaphoreui/charts. */}}
{{- define "texec.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "texec.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "texec.oidcProviders" -}}
{{- $providers := dict -}}
{{- if .Values.oidc.enable -}}
{{- range $id, $values := .Values.oidc.providers -}}
{{- $provider := dict "display_name" $id "return_via_state" true -}}
{{- range $key, $value := omit $values "existingSecret" "clientSecretKey" "clientIdKey" "endpoint" -}}
{{- $_ := set $provider ($key | snakecase) $value -}}
{{- end -}}
{{- with $values.endpoint -}}
{{- $endpoint := dict -}}
{{- range $key, $value := . -}}
{{- $_ := set $endpoint ($key | snakecase) $value -}}
{{- end -}}
{{- $_ := set $provider "endpoint" $endpoint -}}
{{- end -}}
{{- if not $values.redirectUrl -}}
{{- $_ := set $provider "redirect_url" (printf "%s/api/auth/oidc/%s/redirect" (trimSuffix "/" $.Values.general.host) $id) -}}
{{- end -}}
{{- $_ := set $provider "client_secret_file" (printf "/etc/texec/oidc/%s/client-secret" $id) -}}
{{- if $values.clientIdKey -}}
{{- $_ := set $provider "client_id_file" (printf "/etc/texec/oidc/%s/client-id" $id) -}}
{{- end -}}
{{- $_ := set $providers $id $provider -}}
{{- end -}}
{{- end -}}
{{- $providers | toJson -}}
{{- end -}}
