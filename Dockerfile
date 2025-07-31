# ---- Stage 1: Builder ----
# Use Alpine base image. Install build-essentials for any C extensions in Python packages.
FROM python:3.12-alpine AS builder

RUN apk add --no-cache build-base

# Create and activate a virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy and install requirements
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt


# ---- Stage 2: Final Image ----
FROM python:3.12-alpine

# Install su-exec, the lightweight equivalent of gosu for Alpine
RUN apk add --no-cache su-exec

# Copy the virtual environment from the builder stage
COPY --from=builder /opt/venv /opt/venv

# Set the working directory
WORKDIR /app

# Copy the application source code and the new entrypoint script
COPY . /app
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

# Add the venv to the PATH
ENV PATH="/opt/venv/bin:$PATH"

# Set the entrypoint to our script
ENTRYPOINT ["entrypoint.sh"]

# Set the default command to be executed by the entrypoint
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "1", "app:app"]
