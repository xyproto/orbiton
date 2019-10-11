#!/usr/bin/env bash
cd "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
vagrant up --provision
