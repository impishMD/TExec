# Установка в Linux

[English](../en/linux.md) | **Русский**

Используйте DEB/RPM для своей архитектуры (amd64 или arm64). Соберите пакеты командой
`make release-snapshot` по [инструкции выпуска](releasing.md) либо выберите доступный
пакет TExec в [GitHub Releases](https://github.com/impishMD/TExec/releases).
Перед установкой проверьте его по прилагаемому `checksums.txt`.

```sh
sudo apt install ./texec_<version>_linux_amd64.deb
# Или: sudo dnf install ./texec_<version>_linux_amd64.rpm
```

Пакет устанавливает `/usr/bin/texec`, `texec.service`, пользователя/группу `texec`
и каталоги `/etc/texec`, `/var/lib/texec`. Служба не запускается без настройки.
Ansible и другие используемые инструменты установите отдельно.

```sh
cd /etc/texec
sudo -u texec /usr/bin/texec setup
# Для SQLite: /var/lib/texec/database.sqlite; каталог конфигурации: /etc/texec.
sudo systemctl enable --now texec
sudo journalctl -u texec -f
```

Служба читает `/etc/texec/config.json` и необязательный `/etc/texec/env`.
Перед обновлением сделайте резервную копию, установите новый пакет и выполните
`sudo systemctl restart texec`. До удаления пользователя или данных остановите службу.
Установка/удаление пакета не предназначены для удаления конфигурации или базы.
