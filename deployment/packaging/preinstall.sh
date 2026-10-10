#!/bin/sh
set -eu
getent group texec >/dev/null || groupadd --system texec
id texec >/dev/null 2>&1 || useradd --system --gid texec --home-dir /var/lib/texec --shell /usr/sbin/nologin texec
install -d -m 0750 -o texec -g texec /var/lib/texec /etc/texec
