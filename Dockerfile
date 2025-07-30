# Dockerfile

# ---- Stage 1: Builder ----
FROM python:3.12-slim AS builder

# Create and activate a virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy only the requirements file to leverage Docker cache
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt


# ---- Stage 2: Final Image ----
FROM python:3.12-slim

# Install gosu for privilege handling
RUN apt-get update && apt-get install -y gosu && rm -rf /var/lib/apt/lists/*

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
CMD ["python", "app.py"]
