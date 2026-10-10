#!/usr/bin/env bash
set -e

HERE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

mkdir -p /etc/texec
cp ${HERE}/../texec.service /etc/systemd/system
cp ${HERE}/../env /etc/texec/env
systemctl daemon-reload
systemctl start texec.service