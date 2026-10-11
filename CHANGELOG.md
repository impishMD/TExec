# Changelog

## Unreleased

- Fix overlapping task status bars by keeping a single ordered point per day, including zero counts for missing statuses.
- Include failed and stopped tasks without a start time in statistics, using creation dates consistently for filtering and grouping.
- Improve chart labels and stopped-task contrast in dark mode; daily tooltips show a date without a misleading time.

## v1.2.0

- Customize project task success and error notifications per channel, starting with Telegram.
- Preview messages with task variables, completion date/time and a shortened error summary.
- Reset each template to the current interface language; saved text keeps its language across account changes.
- Configure Telegram through General and Message templates tabs with one save and test action; disabled channels also disable their template controls.
- Preserve channel message templates in project backups and imports without exporting notification credentials.
- Helm chart 1.2.0 deploys TExec v1.2.0.

[English release notes](release-notes/en/v1.2.0.md) · [Русский](release-notes/ru/v1.2.0.md)

## v1.1.0

- Workflow notification nodes and approval alerts with Telegram delivery, message variables, previews and delivery history.
- Persistent notification queue with retries and configurable delivery failure behavior.
- Improved workflow editor and run headers: readable names, statuses, revision numbers and counters.
- Repository file suggestions for Ansible inventories, including YAML, INI and extensionless files, with manual path entry.
- Releases from `release/*` branches reuse successful CI for the tagged commit.
- Helm chart 1.1.0 deploys TExec v1.1.0.

[English release notes](release-notes/en/v1.1.0.md) · [Русский](release-notes/ru/v1.1.0.md)

## v1.0.0

First TExec release. TExec is an independent fork of Semaphore UI.

- Automation for Ansible, Terraform, OpenTofu, Terragrunt and scripts.
- Projects, inventories, repositories, credentials, templates, workflows and schedules.
- Personal and project API tokens, role-based access, OIDC, LDAP and two-factor authentication.
- Execution logs, audit events and notification settings.
- Linux/macOS binaries, DEB/RPM packages, multi-platform containers and Helm chart 1.0.0.

[English release notes](release-notes/en/v1.0.0.md) · [Русский](release-notes/ru/v1.0.0.md)
