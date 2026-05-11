#!/bin/sh
set -e

echo "Running migrations..."
goose up

echo "Starting VaultHook..."
exec ./vaulthook
