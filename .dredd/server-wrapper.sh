#!/usr/bin/env bash

export TEXEC_MAX_TASKS_PER_TEMPLATE=300
export TEXEC_APPS='{"ansible": {}}'
export TEXEC_PORT=58427

texec=./texec
[[ -x "$texec" ]] || texec=./bin/texec

exec "$texec" server --config .dredd/config.json
