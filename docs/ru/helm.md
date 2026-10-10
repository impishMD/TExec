# Kubernetes и Helm

[English](../en/helm.md) | **Русский**

[Чарт TExec](../../charts/texec/README.md) основан на [Semaphore UI Charts](https://github.com/semaphoreui/charts)
и разворачивает один сервер, постоянный том и Service. Поддерживаются OIDC, Ingress/TLS,
собственные CA-сертификаты и внешняя PostgreSQL или MySQL.
Нужны Kubernetes 1.26+, Helm 3 или 4, StorageClass либо существующий PVC.

Образ сервера содержит инструменты автоматизации. Задачи выполняются в pod сервера
или на отдельно настроенных удалённых исполнителях. Чарт не устанавливает исполнителей,
базу данных, Redis или Ingress-контроллер. Несколько реплик сервера не поддерживаются.

## Реквизиты

Создайте Secret один раз в том же namespace, что и приложение. Чарт использует ссылку
на существующий Secret: пароли не попадают в values и манифесты Helm-релиза.
Для GitOps создавайте Secret через принятую у вас систему управления секретами.

При ручной установке:

```sh
kubectl create namespace texec
umask 077
bootstrap_dir=$(mktemp -d)
openssl rand -hex 24 | tr -d '\n' > "$bootstrap_dir/admin-password"
openssl rand -base64 32 | tr -d '\n' > "$bootstrap_dir/encryption-key"
kubectl -n texec create secret generic texec-credentials \
  --from-file=TEXEC_ADMIN_PASSWORD="$bootstrap_dir/admin-password" \
  --from-file=TEXEC_ACCESS_KEY_ENCRYPTION="$bootstrap_dir/encryption-key"
# Сохраните оба файла в менеджере паролей/резервной копии перед удалением временных файлов.
rm -r "$bootstrap_dir"
```

Ключ шифрования должен оставаться одинаковым при замене pod и обновлении Helm-релиза.
Без него TExec не сможет расшифровать сохранённые реквизиты. Пароль администратора
используется только при первоначальной настройке; далее меняйте его через TExec.

## Установка

```yaml
# texec-values.yaml
secrets:
  existingSecret: texec-credentials
persistence:
  size: 10Gi
```

```sh
helm repo add texec https://impishmd.github.io/TExec
helm repo update
helm upgrade --install texec texec/texec --version 1.0.0 \
  --namespace texec -f texec-values.yaml --wait --timeout 5m
helm test texec -n texec --logs
kubectl -n texec port-forward service/texec 3000:3000
```

Откройте <http://localhost:3000>. Логин — `admin`, пароль — сохранённый выше.
Для установки из исходников замените `texec/texec --version 1.0.0` на `./charts/texec`.
Для собственного образа укажите `image.repository` и `image.tag`.

По умолчанию используется `ghcr.io/impishmd/texec:v1.0.0`. Для Docker Hub задайте
`image.repository: impishmd/texec`. `image.digest` позволяет закрепить образ по `sha256:…`
и имеет приоритет над тегом. Для закрытого реестра укажите `image.pullSecrets` в namespace релиза.

## Ingress и TLS

```yaml
general:
  host: https://tasks.example.com/texec
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: tasks.example.com
      paths:
        - path: /texec
          pathType: Prefix
  tls:
    - secretName: texec-tls
      hosts: [tasks.example.com]
```

Создайте TLS Secret отдельно или настройте выпуск сертификата через `ingress.annotations`.
Контроллер должен сохранять путь запроса и поддерживать WebSocket.
Для установки в корне домена используйте путь `/` и уберите `/texec` из `general.host`.
Проверки здоровья и `helm test` учитывают подпуть автоматически. TLS завершается на Ingress;
Service обращается к pod по HTTP, порт 3000.

## OpenID Connect (OIDC)

Создайте клиент у провайдера авторизации и задайте callback URL:
`https://tasks.example.com/texec/api/auth/oidc/keycloak/redirect`.
Поместите client secret в Kubernetes Secret (для GitOps используйте свой менеджер секретов):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: texec-oidc
  namespace: texec
type: Opaque
stringData:
  client-secret: "replace-with-client-secret"
```

```yaml
# texec-values.yaml (OIDC)
general:
  host: https://tasks.example.com/texec
oidc:
  enable: true
  providers:
    keycloak:
      displayName: Keycloak
      providerUrl: https://sso.example.com/realms/main
      clientId: texec
      existingSecret: texec-oidc
      clientSecretKey: client-secret
      scopes: [openid, profile, email]
      usernameClaim: preferred_username
      nameClaim: name
      emailClaim: email
```

В `providerUrl` указывается issuer URL без `/.well-known/openid-configuration`:
метаданные discovery приложение загружает само. `general.host` задаёт публичный callback,
включая подпуть. Такой же путь должен быть настроен в Ingress. Поле `redirectUrl`
позволяет указать callback явно.

Для нескольких провайдеров добавьте записи в `oidc.providers`. ID провайдера входит в
идентичность пользователя и callback URL; после начала использования его не меняйте.
Чтобы читать client ID из того же Secret, замените `clientId` на `clientIdKey: client-id`.
Client secret читается из смонтированного файла и не попадает в ConfigMap или values Helm.
Отсутствующий Secret или ключ останавливает запуск pod.

После проверки SSO параметр `general.passwordLoginDisable: true` отключает вход по паролю.
`oidc.enable: false` отключает провайдеры из настроек чарта. Обновлённый client secret
доставляется Kubernetes в файловый том и читается при следующем входе; `subPath` не используется.

Без discovery вместо `providerUrl` задайте `endpoint` с `issuerUrl`, `authUrl`, `tokenUrl`,
`jwksUrl` и, при необходимости, `userInfoUrl` / `algorithms`. Дополнительно доступны
`returnViaState` (по умолчанию `true`), `requireVerifiedEmail`, `order`, `color` и `icon`.

## Сертификаты и параметры pod

Собственный CA для провайдера авторизации, Git-сервера или другого HTTPS-адреса задаётся
через `customCertificates.existingConfigMap` либо `customCertificates.existingSecret`:

```yaml
customCertificates:
  enabled: true
  existingConfigMap: internal-ca
  key: ca.crt
serviceAccount:
  create: true
  annotations: {}
extraEnvSecrets:
  TEXEC_EMAIL_PASSWORD:
    secret: smtp-credentials
    key: password
```

Чарт добавляет CA к системному набору доверенных сертификатов образа. После изменения
Secret/ConfigMap с CA перезапустите Deployment. ServiceAccount создаётся чартом либо
выбирается через `serviceAccount.create: false` и `serviceAccount.name`.
Автоматическое монтирование токена Kubernetes API отключено.

Поддерживаются параметры upstream-чарта `dnsConfig`, `labels`, `annotations`,
`extraInitContainers`, `extraSidecarContainers`, `envFromSecrets` и `envFromConfigMaps`.
`extraEnvVariables` содержит несекретные значения, `extraEnvSecrets` — ссылки на отдельные
ключи Secret. `config.forwarded_env_vars` задаёт имена env, передаваемых в процессы задач.

## Внешняя база данных

Добавьте `TEXEC_DB_USER` и `TEXEC_DB_PASS` в `texec-credentials`, создайте базу
и выдайте пользователю права на создание и изменение её схемы. Выберите БД при первой установке:

```yaml
database:
  type: postgres
  host: postgres.database.svc.cluster.local
  port: 5432
  name: texec
  options:
    sslmode: require
```

Для MySQL используйте `type: mysql` и порт `3306`; параметры TLS описаны в
[справочнике конфигурации](../en/reference/configuration.md). Значения `options` — строки.
Не добавляйте порт в `host`. Реквизиты подключения остаются в Secret.
Изменение адреса или типа базы не переносит существующие данные.
Для отдельного Secret задайте `database.existingSecret`; имена его ключей задаются
через `database.usernameKey` и `database.passwordKey`. Аналогично `admin.existingSecret`
и `admin.passwordKey` выбирают Secret с начальным паролем администратора.

PVC нужен и с внешней БД: в нём хранится `config/config.json`, включая ключи подписи сессий.
Сертификаты CA и другие файлы можно подключить через `extraVolumes` и `extraVolumeMounts`.
`extraEnvVariables` добавляет несекретные строковые параметры, `extraEnvFrom` — ссылки на дополнительные
Secret/ConfigMap. Для переменных, которыми управляет чарт, используйте отдельные поля values.

## Хранение данных и обновление

PVC подключён в `/var/lib/texec`. SQLite использует `database.sqlite`, конфигурация —
`config/config.json`. Выберите StorageClass, подходящий для работы базы данных.
`persistence.storageClass: null` выбирает класс по умолчанию; `""` — отсутствие класса.
`persistence.existingClaim` подключает готовый PVC из namespace релиза.

Перед обновлением прекратите запуск новых задач, дождитесь завершения текущих и сохраните
базу, конфигурацию с PVC и Secret с ключом шифрования. SQLite копируйте при остановленном
сервере либо через согласованную резервную копию/снимок. Обновление использует `Recreate`
и вызывает короткий перерыв в доступности.
Миграции БД выполняются при запуске; `helm rollback` не откатывает схему базы.
Если старая версия не поддерживает новую схему, восстанавливайте совместимую резервную копию.

Изменения конфигурации в values автоматически перезапускают pod. Обновление содержимого
внешнего Secret само по себе не вызывает перезапуск. После изменения реквизитов:

```sh
kubectl -n texec rollout restart deployment/texec
kubectl -n texec rollout status deployment/texec
```

При удалении Helm-релиза PVC сохраняется. Он также защищён от удаления/prune в Argo CD.
Удаление namespace или авария хранилища всё равно могут уничтожить данные; это не замена
резервным копиям. Для повторной установки с сохранённым томом:

```sh
helm upgrade --install texec texec/texec --version 1.0.0 -n texec \
  -f texec-values.yaml --set persistence.existingClaim=texec --wait
```

При импорте файлов сохраните владельца UID/GID 1001 или включите
`volumePermissions.enabled: true`: init-контейнер исправит права от root.
Оставляйте его выключенным, если драйвер учитывает `fsGroup` или политика кластера запрещает root.

Локальные задачи делят CPU и память с сервером; задавайте `resources` под свою нагрузку.
Временные файлы задач и инструменты, установленные вне PVC, не сохраняются при замене pod.
Для вынесения нагрузки используйте удалённых исполнителей.

## Argo CD

Сначала создайте Secret с реквизитами. Пример Application:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: texec
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://impishmd.github.io/TExec
    chart: texec
    targetRevision: 1.0.0
    helm:
      releaseName: texec
      valuesObject:
        secrets:
          existingSecret: texec-credentials
  destination:
    server: https://kubernetes.default.svc
    namespace: texec
  syncPolicy:
    syncOptions: [CreateNamespace=true]
```

Полный набор параметров: [values.yaml](../../charts/texec/values.yaml).
Справка: [репозитории Helm](https://helm.sh/docs/topics/chart_repository/),
[постоянные тома Kubernetes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).
