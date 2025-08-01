#!/bin/sh
set -e

# Use PUID/PGID from environment or default to 1000
USER_ID=${PUID:-1000}
GROUP_ID=${PGID:-1000}

echo "Starting with UID: $USER_ID, GID: $GROUP_ID"

# Create a group with a specific GID (-S for system group)
addgroup -g "$GROUP_ID" -S appgroup

# Create a user with a specific UID (-S for system user, -G to add to group)
adduser -u "$USER_ID" -S -G appgroup -h /app appuser

# Set ownership for the app and cache directories
chown -R appuser:appgroup /app

# Drop privileges and execute the command passed to the entrypoint (CMD in Dockerfile)
exec su-exec appuser "$@"
