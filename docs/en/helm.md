# Kubernetes and Helm

**English** | [Русский](../ru/helm.md)

The [TExec chart](../../charts/texec/README.md), based on [Semaphore UI Charts](https://github.com/semaphoreui/charts),
deploys one server, a persistent volume and a Service. It supports OIDC, Ingress/TLS,
custom CA certificates and external PostgreSQL or MySQL databases.
It requires Kubernetes 1.26+, Helm 3 or 4, and a storage class or an existing PVC.

The server includes automation tools. Tasks can run in the server pod or on separately
configured remote runners. The chart does not install runners, databases, Redis or an
Ingress controller. Multiple server replicas are not supported.

## Credentials

Create credentials once in the same namespace as the release. The chart references an
existing Secret, so credentials do not appear in Helm values or release manifests.
For GitOps, provision this Secret using your existing secret-management system.

For a manual installation:

```sh
kubectl create namespace texec
umask 077
bootstrap_dir=$(mktemp -d)
openssl rand -hex 24 | tr -d '\n' > "$bootstrap_dir/admin-password"
openssl rand -base64 32 | tr -d '\n' > "$bootstrap_dir/encryption-key"
kubectl -n texec create secret generic texec-credentials \
  --from-file=TEXEC_ADMIN_PASSWORD="$bootstrap_dir/admin-password" \
  --from-file=TEXEC_ACCESS_KEY_ENCRYPTION="$bootstrap_dir/encryption-key"
# Store both files in your password manager/backup before removing the temporary copy.
rm -r "$bootstrap_dir"
```

Keep the encryption key unchanged across pod replacement and Helm upgrades. Losing it
prevents TExec from decrypting saved credentials. The administrator password is used
at first setup only; change it later through TExec.

## Install

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
helm upgrade --install texec texec/texec --version 1.2.0 \
  --namespace texec -f texec-values.yaml --wait --timeout 5m
helm test texec -n texec --logs
kubectl -n texec port-forward service/texec 3000:3000
```

Open <http://localhost:3000>. Sign in as `admin` with the password saved above.
To install directly from this repository, use `./charts/texec` instead of
`texec/texec --version 1.2.0`. For a custom image, set `image.repository` and `image.tag`.

The default image is `ghcr.io/impishmd/texec:v1.2.0`. You can use Docker Hub by setting
`image.repository: impishmd/texec`, or pin `image.digest` to a `sha256:…` digest.
Private registries require `image.pullSecrets` in the release namespace.

## Ingress and TLS

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

Create the TLS Secret separately or configure certificate management through
`ingress.annotations`. The Ingress controller must preserve the path and support WebSockets.
Use `/` for a domain-root deployment and remove `/texec` from `general.host`.
Probes and `helm test` automatically use the public URL's subpath. TLS terminates at Ingress;
connections from the Service to the pod use HTTP on port 3000.

## OpenID Connect (OIDC)

Create a client in your identity provider. Set its callback URL to
`https://tasks.example.com/texec/api/auth/oidc/keycloak/redirect`.
Store the client secret in a Kubernetes Secret (for GitOps, use your Secret manager):

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

`providerUrl` is the issuer URL, without `/.well-known/openid-configuration`. The application
loads discovery metadata itself. `general.host` controls the public callback URL, including
any subpath. Use the same path in Ingress. `redirectUrl` provides an explicit callback override.

Add entries under `oidc.providers` for multiple providers. Provider IDs become part of account
identities and callback URLs; keep them stable. To store the client ID in the same Secret,
replace `clientId` with `clientIdKey: client-id`. Client secrets remain in mounted files,
not ConfigMaps or Helm values. Missing Secrets or keys prevent pod startup.

After verifying SSO, `general.passwordLoginDisable: true` turns off password sign-in.
`oidc.enable: false` disables chart-configured providers. Secret file rotations propagate
through Kubernetes and are read on the next login; there is no `subPath` mount for OIDC secrets.

Without discovery, use `endpoint` with `issuerUrl`, `authUrl`, `tokenUrl`, `jwksUrl`,
and optional `userInfoUrl` / `algorithms` instead of `providerUrl`. Other provider settings
include `returnViaState` (default `true`), `requireVerifiedEmail`, `order`, `color` and `icon`.

## Certificates and workload configuration

A private CA for an identity provider, Git server or another HTTPS endpoint can be supplied
through `customCertificates.existingConfigMap` or `customCertificates.existingSecret`:

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

The chart appends the supplied CA bundle to the image's trust store. Restart the deployment
after changing the CA Secret/ConfigMap. ServiceAccounts may be created or referenced with
`serviceAccount.create: false` and `serviceAccount.name`; API token automount is disabled.

The upstream chart's `dnsConfig`, `labels`, `annotations`, `extraInitContainers`,
`extraSidecarContainers`, `envFromSecrets` and `envFromConfigMaps` options are supported.
`extraEnvVariables` holds non-secret values; `extraEnvSecrets` maps individual Secret keys.
`config.forwarded_env_vars` lists environment names to forward into task processes.

## External database

Add `TEXEC_DB_USER` and `TEXEC_DB_PASS` to `texec-credentials`, create the database,
and grant the user permission to create/alter its schema. Select the database on first install:

```yaml
database:
  type: postgres
  host: postgres.database.svc.cluster.local
  port: 5432
  name: texec
  options:
    sslmode: require
```

For MySQL use `type: mysql` and port `3306`; configure TLS according to the
[database settings](reference/configuration.md). Values in `options` must be strings.
Do not include the port in `host`. Database credentials stay in the Secret.
Changing `database.type` or the server address does not transfer existing data.

Use `database.existingSecret`, `database.usernameKey` and `database.passwordKey` to select
a separate credentials Secret or different key names. `admin.existingSecret` and
`admin.passwordKey` do the same for the bootstrap administrator.

The PVC remains necessary with an external database: it stores `config/config.json`,
including the session-signing keys. Use `extraVolumes` and `extraVolumeMounts` for CA files
or other required configuration files. `extraEnvVariables` adds non-secret string configuration;
`extraEnvFrom` accepts additional Secret/ConfigMap references. Chart-managed environment
variables must be configured through their dedicated values.

## Storage, updates and backups

The PVC mounts at `/var/lib/texec`. SQLite uses `database.sqlite` and configuration
uses `config/config.json`. Use a storage class suitable for database workloads.
`persistence.storageClass: null` selects the cluster default; `""` requests no class.
`persistence.existingClaim` uses an existing PVC in the release namespace.

Before an upgrade, stop new task submissions, let running tasks finish, and back up the
database, persisted configuration and encryption Secret. For SQLite, stop the server or
use a consistent SQLite backup/snapshot. Updates use `Recreate` and cause a short outage.
Database migrations run on startup; `helm rollback` does not undo database migrations.
Restore a matching database/configuration backup when an older binary cannot use the new schema.

Configuration changes in Helm values roll the pod automatically. Updating the contents of
an external Secret does not; after intentional credential changes, restart the deployment:

```sh
kubectl -n texec rollout restart deployment/texec
kubectl -n texec rollout status deployment/texec
```

A normal uninstall retains the PVC. It is also marked against Argo CD pruning/deletion.
Storage-cluster failure or namespace deletion can still destroy it; retention is not a backup.
To reinstall against the retained volume:

```sh
helm upgrade --install texec texec/texec --version 1.2.0 -n texec \
  -f texec-values.yaml --set persistence.existingClaim=texec --wait
```

When importing old files, preserve UID/GID 1001 ownership or use
`volumePermissions.enabled: true` for a one-time root init container. Keep it disabled
where the storage driver already implements `fsGroup` or cluster policy prohibits root.

Local tasks share the pod's CPU/memory limit; adjust `resources` for your workloads.
Temporary task files and tools installed outside the data PVC are not durable across
pod replacement. Remote runners are useful for keeping execution workloads separate.

## Argo CD

Create the credentials Secret first. A Helm Application can then use:

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
    targetRevision: 1.2.0
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

Full settings: [chart values](../../charts/texec/values.yaml).
Upstream references: [Helm repositories](https://helm.sh/docs/topics/chart_repository/)
and [Kubernetes persistent volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/).
