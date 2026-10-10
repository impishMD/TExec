# Migration to TExec

**English** | [Русский](../ru/migration.md)


## Migration from Semaphore UI

| Old identity | TExec identity |
| --- | --- |
| `semaphore` executable/service/user | `texec` |
| `SEMAPHORE_*` variables | `TEXEC_*` |
| Ansible `semaphore_vars` | `texec_vars` |
| `/etc/semaphore`, `/var/lib/semaphore`, `/tmp/semaphore`, `/opt/semaphore` | Corresponding `/…/texec` paths |
| `semaphore_*` Prometheus metrics | `texec_*` |
| Upstream container images | `impishmd/texec` or `ghcr.io/impishmd/texec` |
| Go module `github.com/semaphoreui/semaphore` | `github.com/impishMD/TExec` |

Old environment prefixes are not aliases.  Existing database names and absolute paths in config files
are not renamed automatically. The new default database name is `texec`.

1. Stop the old service and back up the database, configuration and encryption keys.
2. Test a restored copy with TExec first, especially when coming from another upstream commit.
3. Rename environment variables, service commands, paths and container mounts. Preserve
   the database dialect, database name, encryption key and cookie keys for the existing instance.
4. Update runner binaries/configuration at the same time. Do not mix old and new runners.
5. Update dashboards/alerts to `texec_*` metrics and reconnect clients as necessary.
6. Start TExec and verify login, repository access and a representative task.

TExec applies database migrations at startup. To roll back, restore the database backup
and matching application version together. Verify client authentication and task environment settings.
