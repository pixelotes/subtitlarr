# Dockerfile

# ---- Stage 1: Builder ----
# This stage installs dependencies into a virtual environment.
FROM python:3.12-slim AS builder

# Create and activate a virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy only the requirements file to leverage Docker cache
COPY requirements.txt .

# Install dependencies
# --no-cache-dir reduces image size
RUN pip install --no-cache-dir -r requirements.txt


# ---- Stage 2: Final Image ----
# This stage creates the final, lean production image.
FROM python:3.12-slim

# Copy the virtual environment from the builder stage
COPY --from=builder /opt/venv /opt/venv

# Create a non-privileged user to run the application
RUN adduser --system --group --no-create-home appuser
USER appuser

# Set the working directory
WORKDIR /app

# Copy the application source code
COPY . /app

# Add the venv to the PATH
ENV PATH="/opt/venv/bin:$PATH"

# Set the entrypoint for the container. This makes the container
# behave like an executable for our script.
ENTRYPOINT ["python", "app.py"]
