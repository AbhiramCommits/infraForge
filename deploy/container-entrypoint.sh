#!/bin/sh
set -e
mkdir -p /run/sshd
exec /usr/sbin/sshd -D -e
