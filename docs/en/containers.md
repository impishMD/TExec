# Containers and runners

**English** | [Русский](../ru/containers.md)

## Local build

Build and start TExec from source:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --build
```

This uses `texec:dev`. See [Migration](migration.md) when importing an existing Semaphore UI installation.

## Release image naming

| Role | Version tag | Stable alias | Entry point |
| --- | --- | --- | --- |
| Server | `v<version>` | `latest` | Server setup and startup |
| Runner | `v<version>-runner` | `latest-runner` | Remote runner |
| Job | `v<version>-job` | `latest-job` | TExec worker and execution tools |
| Helper | `v<version>-helper` | `latest-helper` | Git/SSH helper, `sh` |

The release workflow publishes these tags at `impishmd/texec` and `ghcr.io/impishmd/texec` for
`linux/amd64` and `linux/arm64`. OCI labels link GHCR packages to the repository.
Server and runner use UID 1001; mounted directories must be writable by that UID.

For a published release, use [compose.yaml](../../compose.yaml) with [.env.example](../../.env.example)
and set `TEXEC_VERSION` to the published version. For a local build, include
`compose.dev.yaml` as above.
The default SQLite volumes are `/etc/texec` (configuration) and `/var/lib/texec` (data).
Do not remove volumes when upgrading. The initial admin environment variables create
an account only on first setup. Change an existing password with `texec user change-by-login`.
For PostgreSQL/MySQL, configure `TEXEC_DB_DIALECT`, `TEXEC_DB_HOST`, `TEXEC_DB_USER`,
`TEXEC_DB_PASS` and `TEXEC_DB`. Secret files are supported by variables such as
`TEXEC_DB_PASS_FILE` and `TEXEC_ADMIN_PASSWORD_FILE`.

## Remote runner

Use a published runner image matching the server version, or build it with
`docker build --target runner -t texec:runner-dev .` and use that image below.
Create a runner registration token in the administrator interface and enable
**Management (gear) → General settings → Use remote runners by default**
(or select a runner tag in the template/inventory), then start:

```sh
docker run -d --name texec-runner --restart unless-stopped \
  -e TEXEC_WEB_ROOT=https://texec.example.com \
  -e TEXEC_RUNNER_REGISTRATION_TOKEN="$RUNNER_REGISTRATION_TOKEN" \
  -v texec-runner-data:/var/lib/texec \
  impishmd/texec:latest-runner
```

Keep the runner data volume: it stores the registered runner token.
Configure task execution with either the local or Docker executor.


## Docker task executor

A remote runner configured with `runner.executor.type = docker` starts one disposable
Linux container for each task. The task uses the same preparation and execution code
as local runs: Git checkout, inventory, SSH/host mappings, vault passwords, variables,
logs, commit information, Ansible summaries and Terraform confirmation. It never falls
back to running tools on the runner host.

Build a compatible task image from the current source:

```sh
docker build --target job -t texec:job-dev .
```

The regular server/runner images from the same source also contain the worker. For
local development, `texec:dev` can be used as the job image. Old job images containing
only execution tools are incompatible. Custom images must retain `/usr/local/bin/texec`
with the `task-worker` command and the tools required by their templates; derive them
from the TExec job image of the same version. Image paths, binaries and SSH policy
belong to the task image, not to the host runner configuration.

Create an active project runner in **Project → Runners**, give it a tag such as `docker`,
and register it (or use its already registered token). Start the runner with:

```sh
TEXEC_WEB_ROOT=http://localhost:3000 \
TEXEC_RUNNER_TOKEN_FILE=/path/to/runner-token \
TEXEC_RUNNER_EXECUTOR_TYPE=docker \
TEXEC_RUNNER_DOCKER_IMAGE=texec:job-dev \
TEXEC_RUNNER_DOCKER_PULL_POLICY=never \
texec runner start --no-config
```

Select the same runner tag in the template. An empty **Executor image** uses the runner's
default; a nonempty value overrides it for that template. A local runner rejects templates
with an explicit container image. For Docker Desktop, set `TEXEC_RUNNER_DOCKER_HOST`
to the endpoint printed by `docker context inspect --format '{{.Endpoints.docker.Host}}'`.
The SDK reads `DOCKER_HOST`, `DOCKER_TLS_VERIFY` and `DOCKER_CERT_PATH`, but not Docker CLI
contexts. Explicit TExec host/certificate settings override the SDK environment.

A containerized runner needs access to the Docker daemon, for example a socket mount
and permission to use it. Only the runner receives this access: task containers have
no Docker socket, host directory mounts, server database or runner registration token.
They use the UID from their image. `privileged` is off by default. Access to the daemon
allows control of its containers; assign trusted administrators to this runner host.

Supported settings include network, CPU/memory limits, TLS and image pull policy:
`always`, `if-not-present` (default), or `never`. Set `never` for preloaded private images;
registry login/credential helper integration is not implemented. Configure a Docker
network reachable from the daemon with access to Git, target hosts and any remote
state backend. Inside a task container, `localhost` refers to that container. Existing
local repository paths must be included in the image; host paths are not mounted.

The runner sends a versioned task payload through Docker's attached input stream. Secrets
are not placed in Docker environment metadata or command arguments. Keys and generated
files exist only in the disposable container. Only explicitly forwarded runner environment
variables and `env_vars` are passed to tools, alongside the task/project variables. Job
images are trusted execution environments and can read their task's secrets.

Stopping a task cancels startup/pulling or stops the running container. Losing the runner
connection also terminates the worker. Containers and anonymous volumes are removed after
success, failure or cancellation. There is no persistent workspace/cache or artifact export
in this executor. **Terraform/OpenTofu state needs a persistent remote backend**,
such as the [TExec HTTP backend](terraform-state.md).

### Executor integration checks

These tests use their own labeled containers and fixtures, without an application database:

```sh
docker build --target job -t texec:job-dev .
docker build --build-arg TEXEC_TEST_BASE=texec:job-dev \
  -t texec:docker-fixture services/tasks/docker/testdata
TEXEC_DOCKER_TEST_IMAGE=texec:docker-fixture \
  go test -race -timeout 4m ./services/tasks/docker
```

Set `DOCKER_HOST` as above when needed. The checks cover Git checkout, scripts, Ansible,
Terraform confirmation/rejection, failures, cancellation, parallel tasks, image overrides,
resource limits, credential metadata isolation and container cleanup.
