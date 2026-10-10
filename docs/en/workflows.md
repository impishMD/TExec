# Workflows

**English** | [Русский](../ru/workflows.md)

Open **Project → Workflows**, create a graph, save it and run it. A graph must have
one executable starting node, no cycles and at most 500 nodes / 2,000 connections.
Notes are annotations and cannot have connections.

| Node | Behavior |
| --- | --- |
| Task | Runs a task template from this project with the node's saved parameters |
| Approval | Waits for approval/rejection; an optional timeout in seconds rejects it |
| Delay | Waits the configured positive number of seconds |
| Notification | Sends a message to the project's Telegram chat; its result follows the selected delivery failure policy |
| Note | Displays a comment without running anything |

Connections run on success, failure, or either result. **All** requires every incoming
condition to match; **Any** starts after the first matching result. Unreachable branches
are skipped. Each node runs at most once per run on a single server. The run is failed
if any executed task fails or an approval is rejected, even when a failure branch succeeds.
Stopping the run stops active/queued tasks and cancels pending approvals and delays.

Saving creates an immutable graph revision. A run keeps its original revision when the
graph is edited. Task template definitions themselves remain shared: a node uses the
current definition when it launches. An optional starting version increments per run
and is shared by its nodes. The first task becomes the build reference for deploy nodes.

Project members can view graphs and runs. Editing/deleting requires **Manage project
resources**; running, stopping and resolving approvals requires **Run project tasks**.
Individual template grants do not grant workflow control. Actions are recorded in the audit log.

## Notifications

Configure Telegram in **Project settings → Notifications** first. Add a **Notification** node,
or enable **Notify when approval is required** on an approval node. Delivery uses the saved
project channel, including its inherited global bot token. Credentials and chat IDs are not
copied into the graph.

Messages are plain text, up to 3000 characters, with `{{ variable.name }}` substitutions.
Expressions, commands and HTML are not executed. Supported variables:

| Variables | Values |
| --- | --- |
| `project.name`, `workflow.name` | Project and workflow names |
| `run.id`, `run.started_by`, `run.url` | Run number, initiator and link |
| `date`, `time`, `timezone` | Enqueue time in the selected timezone; approval notifications use the time approval became pending |
| `source.name`, `source.status`, `source.error` | Selected predecessor's name, result and short error excerpt |
| `approval.deadline` | Approval deadline, or “No deadline” |

The default source combines completed immediate predecessors. You can select a specific
upstream node instead. Unfinished and skipped nodes are excluded; missing values become `—`.
The preview uses sample data; the test button sends a clearly marked sample message to the
project chat. Example:

```text
Approval required: {{ workflow.name }} #{{ run.id }}
{{ date }} {{ time }} ({{ timezone }})
Previous step: {{ source.name }} — {{ source.status }}
Error: {{ source.error }}
Approve before: {{ approval.deadline }}
{{ run.url }}
```

`source.error` includes up to three error-marked lines from the last 100 log records,
with length limits. Recognizable password/token patterns and URL credentials are masked;
this cannot guarantee removal of arbitrary secrets printed by commands. Full logs are not
sent. The rendered message is limited to 4000 characters.

Set the public server URL (`web_host` / `TEXEC_WEB_ROOT`) to include run links and a Telegram
link button. Approvals happen in TExec with normal user permission checks; the button only
opens the run page.

The queue and rendered text are persisted. Delivery makes at most three attempts, waiting
10 then 20 seconds between retries. Interrupted attempts can be reclaimed after one minute.
The run page shows delivery status, attempt count and errors. The default failure policy
records the error but continues through the node's success branch. **Fail the step** uses
the failure branch and fails the run. Approval notification failures never resolve approvals.
Stopping the run, resolving approval or reaching its deadline cancels pending delivery.

Normal operation creates one delivery record per node/run. Telegram has no idempotent send
operation: a crash after Telegram accepts a message but before the database records success
can cause a duplicate after recovery. Editing a graph does not alter queued messages.
Project export includes notification configuration; a full database backup also preserves
the queue and delivery history.

## Ansible values between nodes

Use [`ansible.builtin.set_stats`](https://docs.ansible.com/projects/ansible/latest/collections/ansible/builtin/set_stats_module.html)
with `per_host: false` to publish JSON values. TExec adds a callback alongside the
configured stdout callback. `aggregate` follows Ansible's accumulation rules within a task.
`no_log` results and per-host stats are excluded. Artifacts are public project data,
visible in task logs/details; do not use them to pass credentials.

```yaml
- ansible.builtin.set_stats:
    data:
      release_revision: "{{ revision }}"
    per_host: false
    aggregate: false
```

Downstream nodes receive the completed ancestors' values as variables. Nested objects
merge; later task IDs win on conflicts. Explicit node variables take precedence, followed
by the normal secret-variable handling. Unrelated sibling branches are excluded. This works
with local tasks and remote runners, including Docker execution. Keep server and
runners on the same version. A task's artifacts must be a JSON object no larger than 1 MiB.

## Restart, history and backups

Approvals and delay deadlines are stored in the database and resume after server restart.
Tasks assigned to remote runners use normal runner reconciliation. Interrupted local tasks
(including tasks not yet assigned before restart) are stopped; their workflows are stopped
without replaying commands. Start a new run after checking any partial external changes.
Use one TExec server to coordinate workflows.

Automatic task retention preserves workflow task records. Individual workflow tasks cannot
be deleted while their run exists, and templates referenced by saved graph revisions cannot
be deleted. Delete a workflow only after its active runs finish/stop: this removes its graphs
and run history while retaining ordinary task logs. Project export/import includes the latest
graph and node parameters with references remapped to the restored project. Use a full database
backup to preserve historical revisions, runs, approvals and timers.
