# Linux installation

**English** | [Русский](../ru/linux.md)

Use a DEB/RPM package for your architecture (amd64 or arm64). Build packages with
`make release-snapshot` following [Releasing](releasing.md), or select an available
TExec package from [GitHub Releases](https://github.com/impishMD/TExec/releases).
Verify it against the accompanying `checksums.txt` before installation.

```sh
sudo apt install ./texec_<version>_linux_amd64.deb
# Or: sudo dnf install ./texec_<version>_linux_amd64.rpm
```

The package installs `/usr/bin/texec`, `texec.service`, system user/group `texec`,
and writable `/etc/texec` and `/var/lib/texec` directories. It does not start an
unconfigured service. Install Ansible and the execution tools you intend to use.

```sh
cd /etc/texec
sudo -u texec /usr/bin/texec setup
# Select /var/lib/texec/database.sqlite for SQLite and /etc/texec for configuration.
sudo systemctl enable --now texec
sudo journalctl -u texec -f
```

The service reads `/etc/texec/config.json` and optional `/etc/texec/env`.
Bind addresses, database settings and public URL are described in [Configuration](configuration.md).
Before upgrading, back up configuration and data. Install the new package and run
`sudo systemctl restart texec`. Remove the service before deleting the account or data;
package installation and removal do not intentionally delete the database/configuration.
