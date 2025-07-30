#!/bin/sh
set -e

# Use PUID/PGID from environment or default to 1000
USER_ID=${PUID:-1000}
GROUP_ID=${PGID:-1000}

echo "Starting with UID: $USER_ID, GID: $GROUP_ID"

# Create a group with a specific GID. --gid is the explicit, non-ambiguous flag.
addgroup --gid "$GROUP_ID" appgroup

# Create a user with a specific UID and GID.
# --system creates a system user.
# --no-create-home and --disabled-password are good practices.
adduser \
    --system \
    --uid "$USER_ID" \
    --gid "$GROUP_ID" \
    --no-create-home \
    --disabled-password \
    appuser

# Set ownership for the app and cache directories
chown -R appuser:appgroup /app

# Execute the command passed to the entrypoint (CMD in Dockerfile)
exec gosu appuser "$@"
