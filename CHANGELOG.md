# Changelog

## Unreleased

- Add ten appearance palettes and three higher-contrast options, each with light and dark variants.
- Configure the global default in General settings and save personal overrides under Account → Appearance.
- Apply palettes to the TExec logo, wordmark and favicon, login, navigation, controls, charts and workflows while keeping the existing light/dark switch.

## v1.3.0

- Added schedule timezone configuration in General settings, stored in the database.
- Environment and configuration-file overrides take precedence over the saved timezone and make the setting read-only, with an explanation.
- Fixed schedule enabled controls and added a label to the schedule state column.
- Fixed daily task statistics: failed and stopped tasks without a start time are counted, and status bars stack correctly.
- Improved chart readability in dark mode and removed misleading times from daily tooltips.
- Corrected the launch prompt label to Workspace for Terraform, OpenTofu and Terragrunt.
- Added lightweight checks for pull requests into dev; full validation runs on release branches.

Upgrade note: without an explicit timezone setting, schedules now use the server's system timezone instead of UTC. In Docker, this is the container's timezone. To preserve UTC scheduling, explicitly configure UTC.

Timezone priority: TEXEC_SCHEDULE_TIMEZONE > schedule.timezone in the configuration file > database setting > system timezone.

Container images are published to Docker Hub and GHCR for linux/amd64 and linux/arm64. Helm chart 1.3.0 is published separately.

[Русский](https://github.com/impishMD/TExec/blob/v1.3.0/release-notes/ru/v1.3.0.md)

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
