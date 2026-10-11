# Changelog

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
