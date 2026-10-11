# Configuration

**English** | [Русский](../ru/configuration.md)

TExec reads JSON/YAML configuration and `TEXEC_*` environment variables. Environment
variables take precedence. Use `texec setup` to generate a configuration file,
then `texec server --config /path/to/config.json`. `--no-config` enables environment-only operation.

| Setting | Purpose |
| --- | --- |
| `TEXEC_DB_DIALECT` | `sqlite`, `postgres` or `mysql` |
| `TEXEC_DB_HOST` | SQLite file or database host |
| `TEXEC_PORT` | HTTP port, default 3000 |
| `TEXEC_WEB_ROOT` | Public URL, including any path prefix |
| `TEXEC_TMP_PATH` | Working directory, default `/tmp/texec` |
| `TEXEC_ACCESS_KEY_ENCRYPTION` | Base64 encryption key for stored credentials |
| `TEXEC_METRICS_ENABLED` | Enable Prometheus metrics |

See the [complete generated reference](reference/configuration.md),
[CLI commands](reference/cli.md) and [schema](../../config.schema.yaml).
Keep the encryption key with backups; replacing it prevents decrypting existing secrets.
Set the public URL when using a reverse proxy. Authentication settings include local
accounts, LDAP and OpenID Connect; configure providers using the reference.

## General server settings

Administrators can open **Management (gear) → General settings** and enable
**Use remote runners by default**. This server-wide setting is stored in the database
and survives container recreation. It defaults to off: untagged tasks run on the server.
When enabled, untagged tasks use active remote runners marked as default; matching
project runners take priority over global runners. An explicit runner tag in a template
or inventory always selects a remote runner, regardless of this setting.

Saving applies to newly created tasks without restarting the server. Tasks already
queued in the running server keep their execution mode, and assigned tasks remain on
their runners. If the server recovers an unassigned task from the database after a restart,
it uses the current default. If matching runners are offline or busy, the task waits.
If no enabled runner matches, dispatch fails. There is no automatic fallback to local execution.

Remote execution is configured through the UI. The setting is included in full database
backups, not project JSON exports.
Automation can use the administrator-only `GET /api/settings` and `PUT /api/settings`
endpoints with `{"use_remote_runner": true}` or `{"use_remote_runner": false}`.

## Project message templates

Open **Dashboard → Settings → Alerts → Telegram → Message templates** to set separate
success and error messages. Each channel has its own templates. Inside Telegram, **General** contains the chat
and bot settings, and **Message templates** contains the editor. Editing is available
when the channel is enabled. A task template's success/error notification suppression
still takes precedence. Workflow node messages and approval requests remain
independent.

Variables: `project.name`, `template.name`, `task.id`, `task.status`,
`task.started_by`, `task.url`, `task.message`, `task.version`, `task.error`,
`date`, `time`, `timezone`. Insert them as `{{ task.id }}` or use the selector.
Messages are plain text: no HTML or executable expressions. Saved text and the language of `task.status` stay unchanged when another user
views the form in a different interface language. There is no separate message
language selector. Date and time refer to task completion in the selected timezone.
Missing values appear as `—`. The author is the user's name or the project API
token's name; absent authors appear as `—`.

`task.error` includes a short excerpt of error-marked log lines, never the entire
log. Known credential patterns are masked; arbitrary secrets cannot be inferred.
Templates are limited to 3000 Unicode characters, rendered messages to 4000
(longer results are truncated).

Preview uses sample data. **Test notification** sends the selected example with
a test prefix using the current form’s chat, bot and template, without saving the
form. Both tabs share one test button and one Save button. The channel must be enabled and configured. **Reset this template** restores
only the selected success/error template in the current account/interface language;
all 15 interface languages are supported. The other template remains unchanged.
**Save** applies the new text and channel settings together. Existing
projects retain legacy formatting until templates are first saved. Settings are
included in project backups and applied on the server, including for remote
runner tasks.

## Telegram alerts

Open **Management (gear) → Alerts → Telegram** to save the default bot token
issued by BotFather. Global settings are available to administrators.

In **Project → Dashboard → Settings**, open the **Alerts → Telegram** tabs.
Enable Telegram, enter the destination Chat ID and save. Telegram starts disabled
for every project. The bot needs permission to send messages to that chat.
Template options that suppress success or failure alerts still apply.

**Use the global bot token** is selected by default. Clear it to save a separate
bot token for this project; select it again to remove the override. Saved tokens
are never returned by the API or displayed. An empty replacement field keeps an
existing token. Use **Remove token** to delete the global token explicitly.
Project alert settings require permission to update the project.

**Test notification** sends a message using the current form values without saving them.
In global settings, enter a separate test Chat ID; it is not stored. In project settings,
the test uses the selected global or project bot and the destination shown in the form.
The result appears below the fields. Sending a test does not run a task.

Channel settings are stored separately in the database, with per-project enablement
and destinations. Tokens use the server's access-key encryption configuration;
keep this key with your database backups. `vault check`, `vault rekey`, and rekey
credential backups include alert tokens. Changes apply without restarting the server.
Project JSON exports include channel message templates, but exclude credentials,
destinations and enablement. Imported channels start disabled.

## Backups

Back up the configuration, encryption keys and database together. Stop a SQLite instance
before copying its database, or use a consistent SQLite backup. For PostgreSQL/MySQL,
use the database's native backup tooling. Project JSON export is useful for transferring
project definitions but does not replace a complete instance backup.

## Runners and access permissions

- **Project → Runners** manages runners belonging to that project. Creation, editing,
  deletion, cache clearing and registration token rotation require **Manage project
  resources**. Members can read the list; runner credentials are not included in it.
  To dispatch tasks remotely, enable **Use remote runners by default** in **General settings** or select a runner
  tag in the template/inventory. Matching project runners take priority over global
  runners; tasks without a tag use runners marked as default.
- **Management → Roles** manages global roles and requires administrator access.
  **Project → Team → Roles** manages roles belonging to that project and requires
  **Manage project users** to make changes. Assign a role from the project's team page.
- **Template → Permissions** grants additional template permissions to a custom
  global or project role. Editing these grants requires **Manage project users**.
  Grants add to the role's existing permissions; they do not revoke them.
- A role's slug is immutable and unique across the instance. Built-in slugs are reserved.
  Remove a role's user assignments and template grants before deleting it.

## Task integrations

- [HashiCorp Vault](secret-storages.md): KV v1/v2 secret references and credential mapping.
- [Terraform/OpenTofu HTTP backend](terraform-state.md): states, history and locks stored in the database.
- [Workflows](workflows.md): task graphs, approvals and execution conditions.
- [Docker execution](containers.md#docker-task-executor): task isolation in containers.

## Ansible summaries

Ansible tasks store host recap counters, task errors and execution phases in the
main database. Open a completed task and select **Summary**. This works for local
execution and remote runners. Host counters come from `PLAY RECAP`; ignored/rescued
errors remain in the error list even when the final host status is successful.

The parser supports Ansible's default text callback, including colored output and
multiline JSON/YAML failure details. It preserves Ansible's `no_log` censorship.
Custom stdout callbacks may not produce a compatible recap. Runs ending before a
recap show an empty-state explanation. Error previews are
limited to 1,000 characters; the full emitted output remains in the task log.

Deleting a task deletes its summary. Instance database backups include summaries and task logs;
project JSON exports do not include execution history.

## Deployment limits

Run one TExec server with local, remote or Docker task execution. Authentication supports
local accounts, LDAP, OpenID Connect and TOTP. Audit events are stored in the database.
