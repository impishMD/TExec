#!/usr/bin/env bash
set -e

systemctl stop texec.service
systemctl disable texec.service
rm /etc/systemd/system/texec.service
rm -rf /etc/texec